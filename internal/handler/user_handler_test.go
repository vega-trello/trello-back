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
	registerFunc            func(ctx context.Context, username, password string) (*model.User, error)
	loginFunc               func(ctx context.Context, username, password string) (*model.User, error)
	loginSSOFunc            func(ctx context.Context, provider, extID, username string, metadata json.RawMessage) (*model.User, error)
	getSelfProfileFunc      func(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error)
	updateSelfProfileFunc   func(ctx context.Context, userUUID uuid.UUID, newUsername *string, newPassword *string) (*model.SelfUser, error)
	getOtherUserProfileFunc func(ctx context.Context, targetUserUUID uuid.UUID) (*model.User, error)
	logoutFunc              func(ctx context.Context, userUUID uuid.UUID) error
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
func (m *mockUserService) GetSelfProfile(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error) {
	if m.getSelfProfileFunc != nil {
		return m.getSelfProfileFunc(ctx, userUUID)
	}
	return nil, nil
}

func (m *mockUserService) UpdateSelfProfile(ctx context.Context, userUUID uuid.UUID, newUsername *string, newPassword *string) (*model.SelfUser, error) {
	if m.updateSelfProfileFunc != nil {
		return m.updateSelfProfileFunc(ctx, userUUID, newUsername, newPassword)
	}
	return nil, nil
}
func (m *mockUserService) GetOtherUserProfile(ctx context.Context, targetUserUUID uuid.UUID) (*model.User, error) {
	if m.getOtherUserProfileFunc != nil {
		return m.getOtherUserProfileFunc(ctx, targetUserUUID)
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

	protected.GET("/self", h.GetSelfProfile)
	protected.PATCH("/self", h.UpdateSelfProfile)
	protected.GET("/user", h.GetOtherUserProfile)

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

			assert.Regexp(t, `^[a-z]+_[a-z]+_\d{1,3}$`, username,
				"username should match format: {adjective}_{noun}_{number}")

			assert.GreaterOrEqual(t, len(username), 1, "username should be at least 1 character")
			assert.LessOrEqual(t, len(username), 32, "username should not exceed 32 characters")

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

func TestHandler_GetSelfProfile_Success(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{
		getSelfProfileFunc: func(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error) {
			return &model.SelfUser{
				User:      model.User{UUID: userUUID, Username: "testuser", UserType: "manual"},
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")
	req := httptest.NewRequest("GET", "/self", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp dto.SelfUserResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "testuser", resp.Username)
	assert.Equal(t, "manual", resp.UserType)
}

func TestHandler_GetSelfProfile_UserNotFound(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{
		getSelfProfileFunc: func(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error) {
			return nil, repository.ErrUserNotFound
		},
	}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")

	req := httptest.NewRequest("GET", "/self", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetOtherUserProfile_Success(t *testing.T) {
	testUUID := uuid.New()
	targetUUID := uuid.New()

	svc := &mockUserService{
		getOtherUserProfileFunc: func(ctx context.Context, tUUID uuid.UUID) (*model.User, error) {
			assert.Equal(t, targetUUID, tUUID)
			return &model.User{
				UUID:     tUUID,
				Username: "otheruser",
				UserType: "manual",
			}, nil
		},
	}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")

	req := httptest.NewRequest("GET", "/user?userUUID="+targetUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp dto.UserResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "otheruser", resp.Username)
	assert.Equal(t, "manual", resp.UserType)
	assert.Equal(t, targetUUID, resp.UUID)
}

func TestHandler_GetOtherUserProfile_MissingUserUUID(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")

	req := httptest.NewRequest("GET", "/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "missing_param", errResp.Error)
}

func TestHandler_GetOtherUserProfile_InvalidUserUUID(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")

	req := httptest.NewRequest("GET", "/user?userUUID=not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	// 🔹 Унифицированный код ошибки
	assert.Equal(t, "invalid_param", errResp.Error)
}

func TestHandler_GetOtherUserProfile_NotFound(t *testing.T) {
	testUUID := uuid.New()
	targetUUID := uuid.New()
	svc := &mockUserService{
		getOtherUserProfileFunc: func(ctx context.Context, tUUID uuid.UUID) (*model.User, error) {
			return nil, repository.ErrUserNotFound
		},
	}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")

	req := httptest.NewRequest("GET", "/user?userUUID="+targetUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_UpdateSelfProfile_Success_ChangeUsernameOnly(t *testing.T) {
	testUUID := uuid.New()
	newUsername := "newname"

	svc := &mockUserService{
		updateSelfProfileFunc: func(ctx context.Context, userUUID uuid.UUID, username *string, password *string) (*model.SelfUser, error) {
			assert.Equal(t, testUUID, userUUID)
			assert.NotNil(t, username)
			assert.Equal(t, newUsername, *username)
			assert.Nil(t, password)

			return &model.SelfUser{
				User:      model.User{UUID: userUUID, Username: newUsername, UserType: "manual"},
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")

	body := bytes.NewBufferString(`{"username":"newname"}`)
	req := httptest.NewRequest("PATCH", "/self", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp dto.SelfUserResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "newname", resp.Username)
}

func TestHandler_UpdateSelfProfile_Success_ChangePasswordOnly(t *testing.T) {
	testUUID := uuid.New()

	svc := &mockUserService{
		updateSelfProfileFunc: func(ctx context.Context, userUUID uuid.UUID, username *string, password *string) (*model.SelfUser, error) {
			assert.Equal(t, testUUID, userUUID)
			assert.Nil(t, username)
			assert.NotNil(t, password)
			assert.NotEmpty(t, *password)

			return &model.SelfUser{
				User:      model.User{UUID: userUUID, Username: "user", UserType: "manual"},
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")

	body := bytes.NewBufferString(`{"password":"NewPass123"}`)
	req := httptest.NewRequest("PATCH", "/self", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateSelfProfile_Success_ChangeBoth(t *testing.T) {
	testUUID := uuid.New()
	newUsername := "newname"

	svc := &mockUserService{
		updateSelfProfileFunc: func(ctx context.Context, userUUID uuid.UUID, username *string, password *string) (*model.SelfUser, error) {
			assert.Equal(t, testUUID, userUUID)
			assert.NotNil(t, username)
			assert.Equal(t, newUsername, *username)
			assert.NotNil(t, password)

			return &model.SelfUser{
				User:      model.User{UUID: userUUID, Username: newUsername, UserType: "manual"},
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")

	body := bytes.NewBufferString(`{"username":"newname","password":"NewPass123"}`)
	req := httptest.NewRequest("PATCH", "/self", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp dto.SelfUserResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "newname", resp.Username)
}

func TestHandler_UpdateSelfProfile_NoFieldsProvided(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest("PATCH", "/self", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "validation_error", errResp.Error)
	assert.Contains(t, errResp.Message, "at least one field")
}

func TestHandler_UpdateSelfProfile_InvalidUsername(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")

	body := bytes.NewBufferString(`{"username":"` + string(make([]byte, 33)) + `"}`)
	req := httptest.NewRequest("PATCH", "/self", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid_request", errResp.Error)
}

func TestHandler_UpdateSelfProfile_PasswordTooShort(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")

	// 🔹 Короткий пароль
	body := bytes.NewBufferString(`{"password":"123"}`)
	req := httptest.NewRequest("PATCH", "/self", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid_request", errResp.Error)
}

func TestHandler_UpdateSelfProfile_UsernameConflict(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{
		updateSelfProfileFunc: func(ctx context.Context, userUUID uuid.UUID, username *string, password *string) (*model.SelfUser, error) {
			return nil, service.ErrUserAlreadyExists
		},
	}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")

	body := bytes.NewBufferString(`{"username":"taken"}`)
	req := httptest.NewRequest("PATCH", "/self", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var errResp dto.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "username_taken", errResp.Error)
}

func TestHandler_UpdateSelfProfile_SSO_CannotChangePassword(t *testing.T) {
	testUUID := uuid.New()
	svc := &mockUserService{
		updateSelfProfileFunc: func(ctx context.Context, userUUID uuid.UUID, username *string, password *string) (*model.SelfUser, error) {
			return nil, service.ErrSSOUserPasswordChange
		},
	}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")
	token := generateTestToken(t, testUUID, "test-secret")

	body := bytes.NewBufferString(`{"password":"NewPass123"}`)
	req := httptest.NewRequest("PATCH", "/self", body)
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

func TestHandler_Unauthorized_AllEndpoints(t *testing.T) {
	svc := &mockUserService{}
	r, _ := setupProtectedTestRouter(t, svc, "test-secret")

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/self"},
		{"PATCH", "/self"},
		{"GET", "/user?userUUID=" + uuid.New().String()},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
