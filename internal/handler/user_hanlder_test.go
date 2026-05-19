//go:build !integration
// +build !integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vega-trello/trello-back/internal/auth"
	dto "github.com/vega-trello/trello-back/internal/dto/user"
	"github.com/vega-trello/trello-back/internal/middleware"
	"github.com/vega-trello/trello-back/internal/model"
	"github.com/vega-trello/trello-back/internal/repository"
	"github.com/vega-trello/trello-back/internal/service"
)

type mockUserService struct {
	registerFunc      func(ctx context.Context, username, password string) (*model.User, error)
	loginFunc         func(ctx context.Context, username, password string) (*model.User, error)
	loginSSOFunc      func(ctx context.Context, provider, extID, username string, metadata json.RawMessage) (*model.User, error)
	getProfileFunc    func(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error)
	updateProfileFunc func(ctx context.Context, userUUID uuid.UUID, oldPass, newName, newPass string) (*model.SelfUser, error)
	logoutFunc        func(ctx context.Context, userUUID uuid.UUID) error
}

func (m *mockUserService) Register(ctx context.Context, username, password string) (*model.User, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, username, password)
	}
	return nil, nil
}
func (m *mockUserService) Login(ctx context.Context, username, password string) (*model.User, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, username, password)
	}
	return nil, nil
}
func (m *mockUserService) LoginBySSO(ctx context.Context, provider, extID, username string, metadata json.RawMessage) (*model.User, error) {
	if m.loginSSOFunc != nil {
		return m.loginSSOFunc(ctx, provider, extID, username, metadata)
	}
	return nil, nil
}
func (m *mockUserService) GetProfile(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error) {
	if m.getProfileFunc != nil {
		return m.getProfileFunc(ctx, userUUID)
	}
	return nil, nil
}
func (m *mockUserService) UpdateProfile(ctx context.Context, userUUID uuid.UUID, oldPass, newName, newPass string) (*model.SelfUser, error) {
	if m.updateProfileFunc != nil {
		return m.updateProfileFunc(ctx, userUUID, oldPass, newName, newPass)
	}
	return nil, nil
}
func (m *mockUserService) Logout(ctx context.Context, userUUID uuid.UUID) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, userUUID)
	}
	return nil
}

type mockJWTManager struct {
	generateFunc func(userUUID uuid.UUID) (string, error)
	parseFunc    func(token string) (uuid.UUID, error)
}

func (m *mockJWTManager) Generate(userUUID uuid.UUID) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(userUUID)
	}
	return "", nil
}
func (m *mockJWTManager) Parse(token string) (uuid.UUID, error) {
	if m.parseFunc != nil {
		return m.parseFunc(token)
	}
	return uuid.Nil, nil
}

func generateTestToken(t *testing.T, userUUID uuid.UUID, secret string) string {
	t.Helper()
	jwtMgr := auth.NewJWTManager(secret, time.Hour)
	token, err := jwtMgr.Generate(userUUID)
	require.NoError(t, err)
	return token
}

func setupPublicTestRouter(t *testing.T, svc *mockUserService, jwtSecret string) (*gin.Engine, *UserHandler) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtMgr := auth.NewJWTManager(jwtSecret, time.Hour)
	h := &UserHandler{
		userService: svc,
		jwtManager:  jwtMgr,
		vegaBaseURL: "http://mock-vega.local",
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}
	r := gin.New()
	r.Use(gin.Recovery())
	authGroup := r.Group("/auth")
	authGroup.POST("/register", h.Register)
	authGroup.POST("/login", h.Login)
	authGroup.POST("/sso/exchange", h.ExchangeSSOToken)
	authGroup.POST("/logout", middleware.Auth(jwtMgr), h.Logout)
	return r, h
}

func setupProtectedTestRouter(t *testing.T, svc *mockUserService, jwtSecret string) (*gin.Engine, *auth.JWTManager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtMgr := auth.NewJWTManager(jwtSecret, time.Hour)
	h := &UserHandler{
		userService: svc,
		jwtManager:  jwtMgr,
		vegaBaseURL: "http://mock-vega.local",
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}
	r := gin.New()
	r.Use(gin.Recovery())
	protected := r.Group("")
	protected.Use(middleware.Auth(jwtMgr))
	protected.GET("/user", h.GetProfile)
	protected.PATCH("/user", h.UpdateProfile)
	return r, jwtMgr
}

func TestHandler_Register_Success(t *testing.T) {
	svc := &mockUserService{
		registerFunc: func(ctx context.Context, username, password string) (*model.User, error) {
			return &model.User{UUID: uuid.New(), Username: username, UserType: "manual"}, nil
		},
	}
	r, _ := setupPublicTestRouter(t, svc, "test-secret")
	body := bytes.NewBufferString(`{"username":"newuser","password":"StrongPass123"}`)
	req := httptest.NewRequest("POST", "/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	var resp dto.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Token)
}

func TestHandler_Register_InvalidJSON(t *testing.T) {
	svc := &mockUserService{}
	r, _ := setupPublicTestRouter(t, svc, "test-secret")
	body := bytes.NewBufferString(`{"username":"ok"}`)
	req := httptest.NewRequest("POST", "/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid_request", errResp.Error)
}

func TestHandler_Register_PasswordTooShort(t *testing.T) {
	svc := &mockUserService{}
	r, _ := setupPublicTestRouter(t, svc, "test-secret")
	body := bytes.NewBufferString(`{"username":"user","password":"123"}`)
	req := httptest.NewRequest("POST", "/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid_request", errResp.Error)
}

func TestHandler_Register_UsernameTaken(t *testing.T) {
	svc := &mockUserService{
		registerFunc: func(ctx context.Context, username, password string) (*model.User, error) {
			return nil, service.ErrUserAlreadyExists
		},
	}
	r, _ := setupPublicTestRouter(t, svc, "test-secret")
	body := bytes.NewBufferString(`{"username":"taken","password":"StrongPass123"}`)
	req := httptest.NewRequest("POST", "/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "username_taken", errResp.Error)
}

func TestHandler_Login_Success(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{
		loginFunc: func(ctx context.Context, username, password string) (*model.User, error) {
			return &model.User{UUID: testUUID, Username: username, UserType: "manual"}, nil
		},
	}
	r, _ := setupPublicTestRouter(t, svc, "test-secret")
	body := bytes.NewBufferString(`{"username":"user","password":"pass"}`)
	req := httptest.NewRequest("POST", "/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp dto.LoginResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEmpty(t, resp.Token)
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	svc := &mockUserService{
		loginFunc: func(ctx context.Context, username, password string) (*model.User, error) {
			return nil, service.ErrInvalidCredentials
		},
	}
	r, _ := setupPublicTestRouter(t, svc, "test-secret")
	body := bytes.NewBufferString(`{"username":"user","password":"wrong"}`)
	req := httptest.NewRequest("POST", "/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid_credentials", errResp.Error)
}

func TestHandler_ExchangeSSOToken_Success(t *testing.T) {
	vegaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"uai": "530", "fir": "Иван", "sir": "Петров", "grn": "КМБО-02-22", "gri": "42",
		})
	}))
	defer vegaServer.Close()

	testUUID := uuid.New()
	svc := &mockUserService{
		loginSSOFunc: func(ctx context.Context, provider, extID, username string, metadata json.RawMessage) (*model.User, error) {
			assert.Equal(t, "vega", provider)
			assert.Equal(t, "530", extID)
			assert.Contains(t, username, "petrov")
			assert.JSONEq(t, `{"fir":"Иван","sir":"Петров","mid":"","grn":"КМБО-02-22","gri":"42"}`, string(metadata))
			return &model.User{UUID: testUUID, Username: username, UserType: "sso"}, nil
		},
	}

	r, h := setupPublicTestRouter(t, svc, "test-secret")
	h.vegaBaseURL = vegaServer.URL + "/authservice.php"

	body := bytes.NewBufferString(`{"token":"external_sso_token_123"}`)
	req := httptest.NewRequest("POST", "/auth/sso/exchange", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp dto.SSOExchangeResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEmpty(t, resp.Token)
}

func TestHandler_ExchangeSSOToken_InvalidVegaResponse(t *testing.T) {
	vegaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer vegaServer.Close()

	svc := &mockUserService{}
	r, h := setupPublicTestRouter(t, svc, "test-secret")
	h.vegaBaseURL = vegaServer.URL + "/authservice.php"

	body := bytes.NewBufferString(`{"token":"bad_token"}`)
	req := httptest.NewRequest("POST", "/auth/sso/exchange", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "sso_provider_error", errResp.Error)
}

func TestHandler_ExchangeSSOToken_ServiceError(t *testing.T) {
	vegaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"uai": "530", "fir": "A", "sir": "B"})
	}))
	defer vegaServer.Close()

	svc := &mockUserService{
		loginSSOFunc: func(ctx context.Context, provider, extID, username string, metadata json.RawMessage) (*model.User, error) {
			return nil, service.ErrUserAlreadyExists
		},
	}

	r, h := setupPublicTestRouter(t, svc, "test-secret")
	h.vegaBaseURL = vegaServer.URL + "/authservice.php"

	body := bytes.NewBufferString(`{"token":"token"}`)
	req := httptest.NewRequest("POST", "/auth/sso/exchange", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "username_taken", errResp.Error)
}

func TestHandler_GetProfile_Success(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{
		getProfileFunc: func(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error) {
			return &model.SelfUser{
				User:      model.User{UUID: userUUID, Username: "testuser", UserType: "manual"},
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")
	req := httptest.NewRequest("GET", "/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp dto.SelfUserResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "testuser", resp.Username)
	assert.Equal(t, "manual", resp.UserType)
}

func TestHandler_GetProfile_UserNotFound(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{
		getProfileFunc: func(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error) {
			return nil, repository.ErrUserNotFound
		},
	}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")
	req := httptest.NewRequest("GET", "/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_UpdateProfile_Success(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{
		updateProfileFunc: func(ctx context.Context, userUUID uuid.UUID, oldPass, newName, newPass string) (*model.SelfUser, error) {
			return &model.SelfUser{
				User:      model.User{UUID: userUUID, Username: newName, UserType: "manual"},
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")
	body := bytes.NewBufferString(`{"old_password":"oldpass","username":"newname","password":""}`)
	req := httptest.NewRequest("PATCH", "/user", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp dto.SelfUserResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "newname", resp.Username)
}

func TestHandler_UpdateProfile_SSO_CannotChangePassword(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{
		updateProfileFunc: func(ctx context.Context, userUUID uuid.UUID, oldPass, newName, newPass string) (*model.SelfUser, error) {
			return nil, service.ErrSSOUserPasswordChange
		},
	}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")
	body := bytes.NewBufferString(`{"old_password":"dummy","username":"","password":"NewPass123"}`)
	req := httptest.NewRequest("PATCH", "/user", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "sso_password_change_forbidden", errResp.Error)
}

func TestHandler_Logout_Success(t *testing.T) {
	testUUID := uuid.New()
	called := false
	svc := &mockUserService{
		logoutFunc: func(ctx context.Context, userUUID uuid.UUID) error {
			called = true
			return nil
		},
	}
	r, _ := setupPublicTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
	assert.Contains(t, w.Body.String(), "logged out successfully")
}

func TestHandler_Register_JWTGenerationFails(t *testing.T) {
	svc := &mockUserService{
		registerFunc: func(ctx context.Context, username, password string) (*model.User, error) {
			return &model.User{UUID: uuid.New(), Username: username}, nil
		},
	}
	jwtMock := &mockJWTManager{
		generateFunc: func(userUUID uuid.UUID) (string, error) {
			return "", errors.New("crypto error")
		},
	}
	h := &UserHandler{
		userService: svc,
		jwtManager:  jwtMock,
		vegaBaseURL: "http://mock-vega.local",
		httpClient:  &http.Client{Timeout: 2 * time.Second},
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/auth/register", h.Register)
	body := bytes.NewBufferString(`{"username":"user","password":"StrongPass123"}`)
	req := httptest.NewRequest("POST", "/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "token_generation_failed", errResp.Error)
}
