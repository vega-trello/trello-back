//go:build !integration
// +build !integration

package middleware

import (
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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vega-trello/trello-back/internal/auth"
	dto "github.com/vega-trello/trello-back/internal/dto/user"
	"github.com/vega-trello/trello-back/internal/service"
)

type mockPermissionChecker struct {
	mock.Mock
}

func (m *mockPermissionChecker) Check(ctx context.Context, projectUUID, userUUID uuid.UUID, requiredPerm string) error {
	args := m.Called(ctx, projectUUID, userUUID, requiredPerm)
	return args.Error(0)
}

func setupTestRouterWithChecker(t *testing.T, checker service.PermissionChecker, requiredPerm string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	r := gin.New()

	r.Use(Auth(jwtMgr))
	r.Use(RequirePermission(checker, requiredPerm))

	r.GET("/projects/:projectUUID/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	return r
}

func generateTestToken(t *testing.T, jwtMgr *auth.JWTManager, userUUID uuid.UUID) string {
	t.Helper()
	token, err := jwtMgr.Generate(userUUID)
	require.NoError(t, err)
	return token
}

func TestRequirePermission_Success_CheckerAllows(t *testing.T) {
	mockChecker := new(mockPermissionChecker)
	r := setupTestRouterWithChecker(t, mockChecker, "manage_tasks")

	projectUUID := uuid.New()
	userUUID := uuid.New()
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	token := generateTestToken(t, jwtMgr, userUUID)

	mockChecker.On("Check", mock.Anything, projectUUID, mock.AnythingOfType("uuid.UUID"), "manage_tasks").
		Return(nil)

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["message"])
	mockChecker.AssertExpectations(t)
}

func TestRequirePermission_Denied_CheckerRejects(t *testing.T) {
	mockChecker := new(mockPermissionChecker)
	r := setupTestRouterWithChecker(t, mockChecker, "manage_roles")

	projectUUID := uuid.New()
	userUUID := uuid.New()
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	token := generateTestToken(t, jwtMgr, userUUID)

	mockChecker.On("Check", mock.Anything, projectUUID, mock.AnythingOfType("uuid.UUID"), "manage_roles").
		Return(service.ErrPermissionDenied)

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var errResp dto.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.Equal(t, "access_denied", errResp.Error)
	assert.Contains(t, errResp.Message, "Insufficient permissions")
	mockChecker.AssertExpectations(t)
}

func TestRequirePermission_Denied_CheckerError(t *testing.T) {
	mockChecker := new(mockPermissionChecker)
	r := setupTestRouterWithChecker(t, mockChecker, "manage_tasks")

	projectUUID := uuid.New()
	userUUID := uuid.New()
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	token := generateTestToken(t, jwtMgr, userUUID)

	dbErr := errors.New("database connection failed")
	mockChecker.On("Check", mock.Anything, projectUUID, mock.AnythingOfType("uuid.UUID"), "manage_tasks").
		Return(dbErr)

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var errResp dto.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.Equal(t, "internal_error", errResp.Error)
	mockChecker.AssertExpectations(t)
}

func TestRequirePermission_NoProjectUUID_InPath(t *testing.T) {
	mockChecker := new(mockPermissionChecker)
	gin.SetMode(gin.TestMode)

	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	r := gin.New()
	r.Use(Auth(jwtMgr))
	r.Use(RequirePermission(mockChecker, "manage_tasks"))

	r.GET("/global-endpoint", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	userUUID := uuid.New()
	token := generateTestToken(t, jwtMgr, userUUID)
	req := httptest.NewRequest("GET", "/global-endpoint", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockChecker.AssertNotCalled(t, "Check")
}

func TestRequirePermission_InvalidProjectUUID_Format(t *testing.T) {
	mockChecker := new(mockPermissionChecker)
	r := setupTestRouterWithChecker(t, mockChecker, "manage_tasks")

	userUUID := uuid.New()
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	token := generateTestToken(t, jwtMgr, userUUID)

	req := httptest.NewRequest("GET", "/projects/not-a-uuid/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp dto.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.Equal(t, "invalid_uuid", errResp.Error)
	mockChecker.AssertNotCalled(t, "Check")
}

func TestRequirePermission_NoUserUUID_InContext(t *testing.T) {
	mockChecker := new(mockPermissionChecker)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RequirePermission(mockChecker, "manage_tasks"))

	projectUUID := uuid.New()
	r.GET("/projects/:projectUUID/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var errResp dto.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.Equal(t, "context_error", errResp.Error)
	mockChecker.AssertNotCalled(t, "Check")
}

func TestAuthAndPermission_Integration_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockChecker := new(mockPermissionChecker)
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	r := gin.New()

	r.Use(Auth(jwtMgr))
	r.Use(RequirePermission(mockChecker, "manage_tasks"))

	projectUUID := uuid.New()
	userUUID := uuid.New()

	r.GET("/projects/:projectUUID/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "access_granted"})
	})

	token, err := jwtMgr.Generate(userUUID)
	require.NoError(t, err)

	mockChecker.On("Check", mock.Anything, projectUUID, userUUID, "manage_tasks").
		Return(nil)

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "access_granted", resp["message"])
	mockChecker.AssertExpectations(t)
}

func TestAuthAndPermission_Integration_AuthFails(t *testing.T) {
	mockChecker := new(mockPermissionChecker)
	gin.SetMode(gin.TestMode)

	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	r := gin.New()
	r.Use(Auth(jwtMgr))
	r.Use(RequirePermission(mockChecker, "manage_tasks"))

	r.GET("/projects/:projectUUID/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should_not_reach"})
	})

	req := httptest.NewRequest("GET", "/projects/"+uuid.New().String()+"/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "malformed_token")
	mockChecker.AssertNotCalled(t, "Check")
}

func TestAuthAndPermission_Integration_PermissionFails(t *testing.T) {
	mockChecker := new(mockPermissionChecker)
	gin.SetMode(gin.TestMode)

	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	r := gin.New()
	r.Use(Auth(jwtMgr))
	r.Use(RequirePermission(mockChecker, "manage_roles"))

	projectUUID := uuid.New()
	userUUID := uuid.New()

	r.GET("/projects/:projectUUID/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should_not_reach"})
	})

	token, err := jwtMgr.Generate(userUUID)
	require.NoError(t, err)

	mockChecker.On("Check", mock.Anything, projectUUID, userUUID, "manage_roles").
		Return(service.ErrPermissionDenied)

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "access_denied")
	mockChecker.AssertExpectations(t)
}

func TestRespondError_Helper(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	respondError(c, http.StatusForbidden, "access_denied", "No permission")

	assert.Equal(t, http.StatusForbidden, c.Writer.Status())

	var resp dto.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "access_denied", resp.Error)
	assert.Equal(t, "No permission", resp.Message)
}
