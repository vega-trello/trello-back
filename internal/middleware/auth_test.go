package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vega-trello/trello-back/internal/auth"
)

func setupTestRouter(t *testing.T, jwtManager *auth.JWTManager) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Auth(jwtManager))

	r.GET("/protected", func(c *gin.Context) {
		userUUID, exists := GetUserUUID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "context_error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_uuid": userUUID.String()})
	})

	return r
}

func TestAuth_MissingHeader(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	r := setupTestRouter(t, jwtMgr)

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing_token")
}

func TestAuth_InvalidFormat(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	r := setupTestRouter(t, jwtMgr)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_token_format")
}

func TestAuth_ExpiredToken(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret", time.Millisecond*10)
	r := setupTestRouter(t, jwtMgr)

	token, _ := jwtMgr.Generate(uuid.New(), []string{"view_project"})
	time.Sleep(time.Millisecond * 20)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "token_expired")
}

func TestAuth_InvalidSignature(t *testing.T) {
	jwtMgr1 := auth.NewJWTManager("secret-1", time.Hour)
	jwtMgr2 := auth.NewJWTManager("secret-2", time.Hour)
	r := setupTestRouter(t, jwtMgr2)

	token, _ := jwtMgr1.Generate(uuid.New(), []string{"view_project"})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_signature")
}

func TestAuth_MalformedToken(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	r := setupTestRouter(t, jwtMgr)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "malformed_token")
}

func TestAuth_Success(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	r := setupTestRouter(t, jwtMgr)

	originalUUID := uuid.New()
	token, err := jwtMgr.Generate(originalUUID, []string{"view_project", "manage_tasks"})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), originalUUID.String())
}

func TestAuth_Success_WithEmptyPermissions(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	r := setupTestRouter(t, jwtMgr)

	originalUUID := uuid.New()
	token, err := jwtMgr.Generate(originalUUID, []string{})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), originalUUID.String())
}

func TestAuth_ClaimsAddedToContext(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Auth(jwtMgr))

	r.GET("/check-claims", func(c *gin.Context) {
		claims, ok := getClaims(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "claims_not_found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user_uuid":   claims.Subject,
			"permissions": claims.Permissions,
		})
	})

	originalUUID := uuid.New()
	expectedPerms := []string{"manage_tasks", "view_project"}
	token, err := jwtMgr.Generate(originalUUID, expectedPerms)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/check-claims", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, originalUUID.String(), resp["user_uuid"])

	permsRaw, ok := resp["permissions"].([]interface{})
	require.True(t, ok, "permissions should be an array")

	perms := make([]string, len(permsRaw))
	for i, v := range permsRaw {
		perms[i] = v.(string)
	}

	assert.ElementsMatch(t, expectedPerms, perms)
}
