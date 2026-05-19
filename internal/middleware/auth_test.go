// internal/middleware/auth_test.go
package middleware

import (
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

	token, _ := jwtMgr.Generate(uuid.New())
	time.Sleep(time.Millisecond * 20) // Ждём истечения

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
	r := setupTestRouter(t, jwtMgr2) // Middleware использует secret-2

	token, _ := jwtMgr1.Generate(uuid.New()) // Токен подписан secret-1

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_signature")
}

func TestAuth_Success(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	r := setupTestRouter(t, jwtMgr)

	originalUUID := uuid.New()
	token, err := jwtMgr.Generate(originalUUID)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), originalUUID.String())
}
