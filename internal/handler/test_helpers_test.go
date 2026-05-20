//go:build !integration
// +build !integration

package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vega-trello/trello-back/internal/auth"
	"github.com/vega-trello/trello-back/internal/middleware"
)

type MockJWTManager struct {
	GenerateFunc func(userUUID uuid.UUID) (string, error)
	ParseFunc    func(token string) (uuid.UUID, error)
}

func (m *MockJWTManager) Generate(userUUID uuid.UUID) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(userUUID)
	}
	return "test.jwt.token", nil
}

func (m *MockJWTManager) Parse(token string) (uuid.UUID, error) {
	if m.ParseFunc != nil {
		return m.ParseFunc(token)
	}
	return uuid.Nil, nil
}

func GenerateTestToken(t *testing.T, userUUID uuid.UUID, secret string) string {
	t.Helper()
	jwtMgr := auth.NewJWTManager(secret, time.Hour)
	token, err := jwtMgr.Generate(userUUID)
	require.NoError(t, err)
	return token
}

func SetupTestRouterWithAuth(t *testing.T, jwtSecret string, registerRoutes func(*gin.RouterGroup)) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	jwtReal := auth.NewJWTManager(jwtSecret, time.Hour)

	r := gin.New()
	r.Use(gin.Recovery())
	protected := r.Group("")
	protected.Use(middleware.Auth(jwtReal))

	registerRoutes(protected)

	return r
}

func InjectUserUUID(c *gin.Context, userUUID uuid.UUID) {
	c.Set(middleware.ContextKeyUserUUID, userUUID)
}

func AssertErrorResponse(t *testing.T, body []byte, expectedCode string, expectedStatus int) {
	t.Helper()
	var resp map[string]string
	require.NoError(t, json.Unmarshal(body, &resp))
	require.Equal(t, expectedCode, resp["error"])
}
