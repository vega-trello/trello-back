//go:build !integration
// +build !integration

package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vega-trello/trello-back/internal/auth"
	dto "github.com/vega-trello/trello-back/internal/dto/user"
)

func setupTestRouterWithAuth(t *testing.T, requiredPerm string, userPerms []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	r := gin.New()

	r.Use(Auth(jwtMgr))

	r.Use(RequirePermission(requiredPerm))

	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	return r
}

func generateTestToken(t *testing.T, jwtMgr *auth.JWTManager, perms []string) string {
	t.Helper()
	token, err := jwtMgr.Generate(uuid.New(), perms)
	require.NoError(t, err)
	return token
}

func TestRequirePermission_Success_HasPermission(t *testing.T) {
	r := setupTestRouterWithAuth(t, "manage_tasks", []string{"view_project", "manage_tasks", "manage_members"})

	token := generateTestToken(t, auth.NewJWTManager("test-secret", time.Hour), []string{"manage_tasks"})
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["message"])
}

func TestRequirePermission_Success_MultiplePermissions(t *testing.T) {
	r := setupTestRouterWithAuth(t, "view_project", []string{"view_project", "manage_tasks", "manage_roles"})

	token := generateTestToken(t, auth.NewJWTManager("test-secret", time.Hour), []string{"view_project", "manage_tasks", "manage_roles"})
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_Denied_MissingPermission(t *testing.T) {
	r := setupTestRouterWithAuth(t, "manage_roles", []string{"view_project", "manage_tasks"})

	token := generateTestToken(t, auth.NewJWTManager("test-secret", time.Hour), []string{"view_project", "manage_tasks"})
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var errResp dto.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.Equal(t, "access_denied", errResp.Error)
	assert.Contains(t, errResp.Message, "Insufficient permissions")
}

func TestRequirePermission_Denied_EmptyPermissions(t *testing.T) {
	r := setupTestRouterWithAuth(t, "manage_tasks", []string{})

	token := generateTestToken(t, auth.NewJWTManager("test-secret", time.Hour), []string{})
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_Denied_NilPermissions(t *testing.T) {
	r := setupTestRouterWithAuth(t, "manage_tasks", nil)

	token := generateTestToken(t, auth.NewJWTManager("test-secret", time.Hour), nil)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_NoClaimsInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(RequirePermission("manage_tasks"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var errResp dto.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.Equal(t, "unauthorized", errResp.Error)
	assert.Contains(t, errResp.Message, "Claims not found")
}

func TestRequirePermission_InvalidClaimsType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set(string(contextKeyClaims), "not-a-claims-object")
		c.Next()
	})
	r.Use(RequirePermission("manage_tasks"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var errResp dto.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.Equal(t, "unauthorized", errResp.Error)
}

func TestClaims_HasPermission_Helper(t *testing.T) {
	claims := &auth.Claims{
		Permissions: []string{"view_project", "manage_tasks", "manage_members"},
	}

	assert.True(t, claims.HasPermission("view_project"))
	assert.True(t, claims.HasPermission("manage_tasks"))
	assert.True(t, claims.HasPermission("manage_members"))
	assert.False(t, claims.HasPermission("manage_roles"))
	assert.False(t, claims.HasPermission("delete_project"))
	assert.False(t, claims.HasPermission(""))
}

func TestClaims_HasPermission_CaseSensitive(t *testing.T) {
	claims := &auth.Claims{
		Permissions: []string{"Manage_Tasks"},
	}

	// 🔹 Проверка чувствительности к регистру
	assert.False(t, claims.HasPermission("manage_tasks"))
	assert.True(t, claims.HasPermission("Manage_Tasks"))
}

func TestClaims_HasPermission_EmptyOrNil(t *testing.T) {
	claimsEmpty := &auth.Claims{Permissions: []string{}}
	assert.False(t, claimsEmpty.HasPermission("manage_tasks"))

	claimsNil := &auth.Claims{Permissions: nil}
	assert.False(t, claimsNil.HasPermission("manage_tasks"))
}

func TestGetClaims_Helper(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	claims, ok := getClaims(c)
	assert.Nil(t, claims)
	assert.False(t, ok)

	expectedClaims := &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: uuid.New().String()},
		Permissions:      []string{"view_project"},
	}
	c.Set(string(contextKeyClaims), expectedClaims)

	claims, ok = getClaims(c)
	assert.True(t, ok)
	assert.Equal(t, expectedClaims, claims)
}

func TestAuthAndPermission_Integration_Success(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Auth(jwtMgr))
	r.Use(RequirePermission("manage_tasks"))

	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "access_granted"})
	})

	token := generateTestToken(t, jwtMgr, []string{"view_project", "manage_tasks"})
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "access_granted", resp["message"])
}

func TestAuthAndPermission_Integration_AuthFails(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Auth(jwtMgr))
	r.Use(RequirePermission("manage_tasks"))

	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should_not_reach"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "malformed_token")
}

func TestAuthAndPermission_Integration_PermissionFails(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Auth(jwtMgr))
	r.Use(RequirePermission("manage_roles"))

	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should_not_reach"})
	})

	token := generateTestToken(t, jwtMgr, []string{"view_project", "manage_tasks"})
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "access_denied")
}
