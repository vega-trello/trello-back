package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/user"
	"github.com/vega-trello/trello-back/internal/middleware"
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

	username := utils.GenerateRandomUsername()

	metadata, _ := json.Marshal(map[string]interface{}{
		"fir": profile.FIR,
		"sir": profile.SIR,
		"mid": profile.MID,
		"grn": profile.GRN,
		"gri": profile.GRI,
	})

	user, err := h.userService.LoginBySSO(c.Request.Context(), "vega", profile.GetUAI(), username, metadata)
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

// GetSelfProfile GET /self
func (h *UserHandler) GetSelfProfile(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	selfUser, err := h.userService.GetSelfProfile(c.Request.Context(), userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromSelfUserModel(selfUser))
}

// UpdateSelfProfile PATCH /self
func (h *UserHandler) UpdateSelfProfile(c *gin.Context) {
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

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	updated, err := h.userService.UpdateSelfProfile(
		c.Request.Context(),
		userUUID,
		req.Username,
		req.Password,
	)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromSelfUserModel(updated))
}

// GetOtherUserProfile GET /user?userUUID={uuid}
func (h *UserHandler) GetOtherUserProfile(c *gin.Context) {
	targetUserUUIDStr := c.Query("userUUID")
	if targetUserUUIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_param", "userUUID query parameter is required")
		return
	}

	targetUserUUID, err := uuid.Parse(targetUserUUIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_param", "userUUID must be a valid UUID")
		return
	}

	user, err := h.userService.GetOtherUserProfile(c.Request.Context(), targetUserUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromUserModel(user))
}

// Logout POST /auth/logout
func (h *UserHandler) Logout(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

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

	if profile.GetUAI() == "" {
		return nil, errors.New("invalid SSO token: missing 'uai'")
	}

	return &profile, nil
}

type VegaProfile struct {
	ISS string `json:"iss"`
	SUB string `json:"sub"`
	AUD string `json:"aud"`
	ROL int    `json:"rol"`
	EXP string `json:"exp"`
	JTI string `json:"jti"`
	UAI int    `json:"uai"`
	GRI int    `json:"gri"`
	GRN string `json:"grn"`
	SIR string `json:"sir"`
	FIR string `json:"fir"`
	MID string `json:"mid"`
}

func (p *VegaProfile) GetUAI() string {
	return strconv.Itoa(p.UAI)
}
