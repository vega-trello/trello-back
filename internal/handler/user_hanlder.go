// internal/handler/user_handler.go
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	dto "github.com/vega-trello/trello-back/internal/dto/user"
	"github.com/vega-trello/trello-back/internal/middleware"
	"github.com/vega-trello/trello-back/internal/repository"
	"github.com/vega-trello/trello-back/internal/service"
	"github.com/vega-trello/trello-back/internal/utils"
)

type UserHandler struct {
	userService UserServiceInterface
	jwtManager  JWTManagerInterface
	vegaBaseURL string
	httpClient  *http.Client
}

func NewUserHandler(
	userService UserServiceInterface,
	jwtManager JWTManagerInterface,
	vegaBaseURL string,
) *UserHandler {
	return &UserHandler{
		userService: userService,
		jwtManager:  jwtManager,
		vegaBaseURL: vegaBaseURL,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Register POST /auth/register
func (h *UserHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	user, err := h.userService.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	token, err := h.jwtManager.Generate(user.UUID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "token_generation_failed", "Failed to generate token")
		return
	}

	c.JSON(http.StatusCreated, dto.LoginResponse{Token: token})
}

// Login POST /auth/login
func (h *UserHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	user, err := h.userService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	token, err := h.jwtManager.Generate(user.UUID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "token_generation_failed", "Failed to generate token")
		return
	}

	c.JSON(http.StatusOK, dto.LoginResponse{Token: token})
}

// ExchangeSSOToken POST /auth/sso/exchange
func (h *UserHandler) ExchangeSSOToken(c *gin.Context) {
	var req dto.SSOExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	profile, err := h.parseVegaToken(c.Request.Context(), req.Token)
	if err != nil {
		respondError(c, http.StatusBadGateway, "sso_provider_error", fmt.Sprintf("SSO provider error: %v", err))
		return
	}

	username := utils.GenerateSSOUsername(profile.FIR, profile.SIR, profile.UAI)

	metadata, _ := json.Marshal(map[string]interface{}{
		"fir": profile.FIR,
		"sir": profile.SIR,
		"mid": profile.MID,
		"grn": profile.GRN,
		"gri": profile.GRI,
	})

	user, err := h.userService.LoginBySSO(c.Request.Context(), "vega", profile.UAI, username, metadata)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	token, err := h.jwtManager.Generate(user.UUID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "token_generation_failed", "Failed to generate token")
		return
	}

	c.JSON(http.StatusOK, dto.SSOExchangeResponse{Token: token})
}

// GetProfile GET /user
func (h *UserHandler) GetProfile(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	selfUser, err := h.userService.GetProfile(c.Request.Context(), userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromSelfUserModel(selfUser))
}

// UpdateProfile PATCH /user
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	updated, err := h.userService.UpdateProfile(c.Request.Context(), userUUID, req.OldPassword, req.Username, req.Password)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromSelfUserModel(updated))
}

// Logout POST /auth/logout
func (h *UserHandler) Logout(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	// Stateless JWT: инвалидация на клиенте
	_ = h.userService.Logout(c.Request.Context(), userUUID)
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

func (h *UserHandler) parseVegaToken(ctx context.Context, token string) (*VegaProfile, error) {
	reqURL := fmt.Sprintf("%s?op=parsetoken&token=%s", h.vegaBaseURL, token)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vega returned status %d", resp.StatusCode)
	}

	var profile VegaProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to parse vega response: %w", err)
	}

	if profile.UAI == "" {
		return nil, errors.New("invalid SSO token: missing 'uai'")
	}

	return &profile, nil
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		respondError(c, http.StatusUnauthorized, "invalid_credentials", err.Error())
	case errors.Is(err, service.ErrPasswordTooShort):
		respondError(c, http.StatusBadRequest, "password_too_short", err.Error())
	case errors.Is(err, service.ErrUserAlreadyExists):
		respondError(c, http.StatusConflict, "username_taken", err.Error())
	case errors.Is(err, service.ErrSSOUserPasswordChange):
		respondError(c, http.StatusForbidden, "sso_password_change_forbidden", err.Error())
	case errors.Is(err, repository.ErrUserNotFound):
		respondError(c, http.StatusNotFound, "user_not_found", "User not found")

	default:
		respondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}

func respondError(c *gin.Context, status int, code string, msg string) {
	c.AbortWithStatusJSON(status, dto.ErrorResponse{
		Error:   code,
		Message: msg,
	})
}

// VegaProfile структура ответа
type VegaProfile struct {
	UAI string `json:"uai"`
	FIR string `json:"fir"`
	SIR string `json:"sir"`
	MID string `json:"mid"`
	GRI string `json:"gri"`
	GRN string `json:"grn"`
}
