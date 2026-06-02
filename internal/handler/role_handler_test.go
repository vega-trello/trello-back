//go:build !integration
// +build !integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dto "github.com/vega-trello/trello-back/internal/dto/role"
	"github.com/vega-trello/trello-back/internal/model"
	"github.com/vega-trello/trello-back/internal/service"
)

type mockRoleService struct {
	createFunc      func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error)
	listFunc        func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Role, error)
	getFunc         func(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) (*model.Role, error)
	updateFunc      func(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error)
	deleteFunc      func(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) error
	permissionsFunc func(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) ([]*model.Permission, error)
}

func (m *mockRoleService) CreateRole(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, projectUUID, userUUID, name, description, permissionIDs)
	}
	return nil, nil
}
func (m *mockRoleService) GetProjectRoles(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Role, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, projectUUID, userUUID)
	}
	return nil, nil
}
func (m *mockRoleService) GetRole(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) (*model.Role, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, projectUUID, roleID, userUUID)
	}
	return nil, nil
}
func (m *mockRoleService) UpdateRole(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, projectUUID, roleID, userUUID, name, description, permissionIDs)
	}
	return nil, nil
}
func (m *mockRoleService) DeleteRole(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, projectUUID, roleID, userUUID)
	}
	return nil
}
func (m *mockRoleService) GetRolePermissions(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) ([]*model.Permission, error) {
	if m.permissionsFunc != nil {
		return m.permissionsFunc(ctx, projectUUID, roleID, userUUID)
	}
	return nil, nil
}

func setupRoleRouter(t *testing.T, roleSvc *mockRoleService, jwtSecret string) *gin.Engine {
	t.Helper()

	h := NewRoleHandler(roleSvc)

	return SetupTestRouterWithAuth(t, jwtSecret, func(rg *gin.RouterGroup) {
		rg.GET("/projects/:projectUUID/roles", h.ListProjectRoles)
		rg.POST("/projects/:projectUUID/roles", h.CreateRole)
		rg.GET("/projects/:projectUUID/roles/:roleID", h.GetRole)
		rg.PATCH("/projects/:projectUUID/roles/:roleID", h.UpdateRole)
		rg.DELETE("/projects/:projectUUID/roles/:roleID", h.DeleteRole)
		rg.GET("/projects/:projectUUID/roles/:roleID/permissions", h.GetRolePermissions)
	})
}

func TestRoleHandler_ListProjectRoles_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()

	roleSvc := &mockRoleService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) ([]*model.Role, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			desc := "Project owner"
			return []*model.Role{
				{
					ID:          1,
					ProjectUUID: &projectUUID,
					Name:        "owner",
					Description: &desc,
				},
			}, nil
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/roles", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var roles []dto.RoleResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &roles))
	require.Len(t, roles, 1)
	assert.Equal(t, "owner", roles[0].Name)
	assert.Equal(t, 1, roles[0].ID)
}

func TestRoleHandler_ListProjectRoles_InvalidUUID(t *testing.T) {
	userUUID := uuid.New()
	roleSvc := &mockRoleService{}
	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/not-a-uuid/roles", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_uuid", http.StatusBadRequest)
}

func TestRoleHandler_ListProjectRoles_AccessDenied(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleSvc := &mockRoleService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) ([]*model.Role, error) {
			return nil, service.ErrAccessDenied
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/roles", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "access_denied", http.StatusForbidden)
}

func TestRoleHandler_CreateRole_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleID := 5
	desc := "Can edit tasks"

	roleSvc := &mockRoleService{
		createFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, "Editor", name)
			assert.Equal(t, desc, *description)
			assert.Equal(t, []int{1, 2, 3}, permissionIDs)
			return &model.Role{
				ID:          roleID,
				ProjectUUID: &projectUUID,
				Name:        name,
				Description: description,
			}, nil
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Editor","description":"Can edit tasks","permission_ids":[1,2,3]}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/roles", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var role dto.RoleResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &role))
	assert.Equal(t, "Editor", role.Name)
	assert.Equal(t, roleID, role.ID)
}

func TestRoleHandler_CreateRole_InvalidName(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleSvc := &mockRoleService{}
	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"","description":"desc","permission_ids":[1]}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/roles", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "validation_error", http.StatusBadRequest)
}

func TestRoleHandler_CreateRole_InvalidDescription(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleSvc := &mockRoleService{}
	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	longDesc := string(make([]byte, 257))
	body := bytes.NewBufferString(`{"name":"Test","description":"` + longDesc + `","permission_ids":[1]}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/roles", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_request", http.StatusBadRequest)
}

func TestRoleHandler_CreateRole_NoPermissions(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()

	roleSvc := &mockRoleService{
		createFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error) {
			if len(permissionIDs) == 0 {
				return nil, service.ErrInvalidPermission
			}
			return &model.Role{
				ID:          1,
				ProjectUUID: &projectUUID,
				Name:        name,
				Description: description,
			}, nil
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Test","description":"desc","permission_ids":[]}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/roles", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "validation_error", http.StatusBadRequest)
}

func TestRoleHandler_GetRole_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleID := 5
	desc := "Project owner"

	roleSvc := &mockRoleService{
		getFunc: func(ctx context.Context, pUUID uuid.UUID, rID int, uUUID uuid.UUID) (*model.Role, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, roleID, rID)
			assert.Equal(t, userUUID, uUUID)
			return &model.Role{
				ID:          roleID,
				ProjectUUID: &projectUUID,
				Name:        "owner",
				Description: &desc,
			}, nil
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/roles/"+strconv.Itoa(roleID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var role dto.RoleResponse
	json.Unmarshal(w.Body.Bytes(), &role)
	assert.Equal(t, "owner", role.Name)
	assert.Equal(t, roleID, role.ID)
}

func TestRoleHandler_GetRole_InvalidRoleID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleSvc := &mockRoleService{}
	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/roles/abc", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_role_id", http.StatusBadRequest)
}

func TestRoleHandler_GetRole_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleID := 999
	roleSvc := &mockRoleService{
		getFunc: func(ctx context.Context, pUUID uuid.UUID, rID int, uUUID uuid.UUID) (*model.Role, error) {
			return nil, service.ErrRoleNotFound
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/roles/"+strconv.Itoa(roleID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "role_not_found", http.StatusNotFound)
}

func TestRoleHandler_GetRole_SystemRoleProtected(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleID := 1
	roleSvc := &mockRoleService{
		getFunc: func(ctx context.Context, pUUID uuid.UUID, rID int, uUUID uuid.UUID) (*model.Role, error) {
			return nil, service.ErrSystemRoleProtected
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/roles/"+strconv.Itoa(roleID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "system_role_protected", http.StatusForbidden)
}

func TestRoleHandler_UpdateRole_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleID := 5
	newDesc := "Updated description"

	roleSvc := &mockRoleService{
		updateFunc: func(ctx context.Context, pUUID uuid.UUID, rID int, uUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, roleID, rID)
			assert.Equal(t, "Editor", name)
			assert.Equal(t, newDesc, *description)
			assert.Equal(t, []int{1, 2}, permissionIDs)
			return &model.Role{
				ID:          roleID,
				ProjectUUID: &projectUUID,
				Name:        name,
				Description: description,
			}, nil
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Editor","description":"Updated description","permission_ids":[1,2]}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/roles/"+strconv.Itoa(roleID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var role dto.RoleResponse
	json.Unmarshal(w.Body.Bytes(), &role)
	assert.Equal(t, "Editor", role.Name)
}

func TestRoleHandler_UpdateRole_SystemRoleProtected(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleID := 1
	roleSvc := &mockRoleService{
		updateFunc: func(ctx context.Context, pUUID uuid.UUID, rID int, uUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error) {
			return nil, service.ErrSystemRoleProtected
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"New Name","description":"desc","permission_ids":[1]}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/roles/"+strconv.Itoa(roleID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "system_role_protected", http.StatusForbidden)
}

func TestRoleHandler_DeleteRole_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleID := 5
	called := false

	roleSvc := &mockRoleService{
		deleteFunc: func(ctx context.Context, pUUID uuid.UUID, rID int, uUUID uuid.UUID) error {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, roleID, rID)
			assert.Equal(t, userUUID, uUUID)
			called = true
			return nil
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/roles/"+strconv.Itoa(roleID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	assert.True(t, called)
}

func TestRoleHandler_DeleteRole_RoleInUse(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleID := 5
	roleSvc := &mockRoleService{
		deleteFunc: func(ctx context.Context, pUUID uuid.UUID, rID int, uUUID uuid.UUID) error {
			return service.ErrRoleInUse
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/roles/"+strconv.Itoa(roleID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "role_in_use", http.StatusConflict)
}

func TestRoleHandler_DeleteRole_SystemRoleProtected(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleID := 1
	roleSvc := &mockRoleService{
		deleteFunc: func(ctx context.Context, pUUID uuid.UUID, rID int, uUUID uuid.UUID) error {
			return service.ErrSystemRoleProtected
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/roles/"+strconv.Itoa(roleID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "system_role_protected", http.StatusForbidden)
}

func TestRoleHandler_GetRolePermissions_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleID := 5

	roleSvc := &mockRoleService{
		permissionsFunc: func(ctx context.Context, pUUID uuid.UUID, rID int, uUUID uuid.UUID) ([]*model.Permission, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, roleID, rID)
			return []*model.Permission{
				{ID: 1, Name: "create_task", Description: "Allow creating tasks"},
				{ID: 2, Name: "edit_task", Description: "Allow editing tasks"},
			}, nil
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/roles/"+strconv.Itoa(roleID)+"/permissions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var perms []dto.PermissionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &perms))
	require.Len(t, perms, 2)
	assert.Equal(t, "create_task", perms[0].Name)
	assert.Equal(t, "edit_task", perms[1].Name)
}

func TestRoleHandler_GetRolePermissions_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	roleID := 999
	roleSvc := &mockRoleService{
		permissionsFunc: func(ctx context.Context, pUUID uuid.UUID, rID int, uUUID uuid.UUID) ([]*model.Permission, error) {
			return nil, service.ErrRoleNotFound
		},
	}

	r := setupRoleRouter(t, roleSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/roles/"+strconv.Itoa(roleID)+"/permissions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "role_not_found", http.StatusNotFound)
}

func TestRoleHandler_Unauthorized_AllEndpoints(t *testing.T) {
	roleSvc := &mockRoleService{}
	r := setupRoleRouter(t, roleSvc, "test-secret")

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/projects/" + uuid.New().String() + "/roles", ""},
		{"POST", "/projects/" + uuid.New().String() + "/roles", `{"name":"Test","description":"desc","permission_ids":[1]}`},
		{"GET", "/projects/" + uuid.New().String() + "/roles/1", ""},
		{"PATCH", "/projects/" + uuid.New().String() + "/roles/1", `{"name":"New","description":"desc","permission_ids":[1]}`},
		{"DELETE", "/projects/" + uuid.New().String() + "/roles/1", ""},
		{"GET", "/projects/" + uuid.New().String() + "/roles/1/permissions", ""},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			var req *http.Request
			if ep.body != "" {
				req = httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(ep.method, ep.path, nil)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
