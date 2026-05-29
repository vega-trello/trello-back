//go:build !integration
// +build !integration

package handler

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
	per "github.com/vega-trello/trello-back/internal/dto/permission"
	dto "github.com/vega-trello/trello-back/internal/dto/user"
	"github.com/vega-trello/trello-back/internal/middleware"
	"github.com/vega-trello/trello-back/internal/model"
)

type mockPermissionService struct {
	mock.Mock
}

var _ PermissionServiceInterface = (*mockPermissionService)(nil)

func (m *mockPermissionService) GetAllPermissions(ctx context.Context) ([]*model.Permission, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Permission), args.Error(1)
}

func setupPermissionTestRouter(t *testing.T, svc *mockPermissionService, jwtSecret string) (*gin.Engine, *PermissionHandler) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtMgr := auth.NewJWTManager(jwtSecret, 1*time.Hour)
	h := NewPermissionHandler(svc)
	r := gin.New()
	r.Use(gin.Recovery())
	protected := r.Group("")
	protected.Use(middleware.Auth(jwtMgr))
	protected.GET("/projects/permissions", h.ListPermissions)
	return r, h
}

func TestPermissionHandler_ListPermissions_Success(t *testing.T) {
	svc := new(mockPermissionService)
	r, _ := setupPermissionTestRouter(t, svc, "test-secret")

	testUUID := uuid.New()
	token := generateTestToken(t, testUUID, "test-secret")

	expected := []*model.Permission{
		{ID: 1, Name: "view_project", Description: "Read-only access"},
		{ID: 6, Name: "manage_tasks", Description: "Task management"},
	}

	svc.On("GetAllPermissions", mock.Anything).Return(expected, nil)

	req := httptest.NewRequest("GET", "/projects/permissions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var responses []per.PermissionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &responses))

	require.Len(t, responses, 2)
	assert.Equal(t, 1, responses[0].ID)
	assert.Equal(t, "view_project", responses[0].Name)
	assert.Equal(t, "Read-only access", responses[0].Description)
	assert.Equal(t, 6, responses[1].ID)
	assert.Equal(t, "manage_tasks", responses[1].Name)

	svc.AssertExpectations(t)
}

func TestPermissionHandler_ListPermissions_Empty(t *testing.T) {
	svc := new(mockPermissionService)
	r, _ := setupPermissionTestRouter(t, svc, "test-secret")

	testUUID := uuid.New()
	token := generateTestToken(t, testUUID, "test-secret")

	svc.On("GetAllPermissions", mock.Anything).Return([]*model.Permission{}, nil)

	req := httptest.NewRequest("GET", "/projects/permissions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]", w.Body.String())
	svc.AssertExpectations(t)
}

func TestPermissionHandler_ListPermissions_ServiceError(t *testing.T) {
	svc := new(mockPermissionService)
	r, _ := setupPermissionTestRouter(t, svc, "test-secret")

	testUUID := uuid.New()
	token := generateTestToken(t, testUUID, "test-secret")

	svc.On("GetAllPermissions", mock.Anything).Return(nil, errors.New("database error"))

	req := httptest.NewRequest("GET", "/projects/permissions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var errResp dto.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.Equal(t, "internal_error", errResp.Error)
	svc.AssertExpectations(t)
}

func TestPermissionHandler_ListPermissions_Unauthorized(t *testing.T) {
	svc := new(mockPermissionService)
	r, _ := setupPermissionTestRouter(t, svc, "test-secret")

	req := httptest.NewRequest("GET", "/projects/permissions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	svc.AssertNotCalled(t, "GetAllPermissions")
}
