//go:build !integration
// +build !integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dto "github.com/vega-trello/trello-back/internal/dto/member"
	"github.com/vega-trello/trello-back/internal/model"
	"github.com/vega-trello/trello-back/internal/service"
)

type mockMemberService struct {
	listFunc       func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*dto.MemberResponse, error)
	getFunc        func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, targetUserUUID uuid.UUID) (*dto.MemberResponse, error)
	addFunc        func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto.CreateMemberRequest) (*model.ProjectMember, error)
	updateRoleFunc func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, targetUserUUID uuid.UUID, req dto.UpdateMemberRequest) (*model.ProjectMember, error)
	removeFunc     func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, targetUserUUID uuid.UUID) error
}

func (m *mockMemberService) GetProjectMembers(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*dto.MemberResponse, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, projectUUID, userUUID)
	}
	return nil, nil
}

func (m *mockMemberService) GetMember(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, targetUserUUID uuid.UUID) (*dto.MemberResponse, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, projectUUID, userUUID, targetUserUUID)
	}
	return nil, nil
}

func (m *mockMemberService) AddMember(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto.CreateMemberRequest) (*model.ProjectMember, error) {
	if m.addFunc != nil {
		return m.addFunc(ctx, projectUUID, userUUID, req)
	}
	return nil, nil
}

func (m *mockMemberService) UpdateMemberRole(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, targetUserUUID uuid.UUID, req dto.UpdateMemberRequest) (*model.ProjectMember, error) {
	if m.updateRoleFunc != nil {
		return m.updateRoleFunc(ctx, projectUUID, userUUID, targetUserUUID, req)
	}
	return nil, nil
}

func (m *mockMemberService) RemoveMember(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, targetUserUUID uuid.UUID) error {
	if m.removeFunc != nil {
		return m.removeFunc(ctx, projectUUID, userUUID, targetUserUUID)
	}
	return nil
}

func setupMemberRouter(t *testing.T, memberSvc *mockMemberService, jwtSecret string) *gin.Engine {
	t.Helper()

	h := NewMemberHandler(memberSvc)

	return SetupTestRouterWithAuth(t, jwtSecret, func(rg *gin.RouterGroup) {
		rg.GET("/projects/:projectUUID/members", h.ListProjectMembers)
		rg.POST("/projects/:projectUUID/members", h.AddMember)
		rg.GET("/projects/:projectUUID/member", h.GetMember)
		rg.PATCH("/projects/:projectUUID/member", h.UpdateMemberRole)
		rg.DELETE("/projects/:projectUUID/member", h.RemoveMember)
	})
}

func TestMemberHandler_ListProjectMembers_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	targetUserUUID := uuid.New()
	now := time.Now()

	memberSvc := &mockMemberService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) ([]*dto.MemberResponse, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			return []*dto.MemberResponse{
				{
					Username:    "owner_user",
					UserUUID:    targetUserUUID.String(),
					ProjectUUID: projectUUID.String(),
					RoleID:      1,
					RoleName:    "owner",
					JoinedAt:    now,
				},
			}, nil
		},
	}

	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/members", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []dto.MemberResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	require.Len(t, resp, 1)
	assert.Equal(t, "owner_user", resp[0].Username)
	assert.Equal(t, "owner", resp[0].RoleName)
	assert.Equal(t, targetUserUUID.String(), resp[0].UserUUID)

}

func TestMemberHandler_ListProjectMembers_Empty_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()

	memberSvc := &mockMemberService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) ([]*dto.MemberResponse, error) {
			return []*dto.MemberResponse{}, nil // пустой слайс
		},
	}

	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/members", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []dto.MemberResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp)
	assert.Equal(t, "[]", w.Body.String())
}

func TestMemberHandler_ListProjectMembers_InvalidUUID(t *testing.T) {
	userUUID := uuid.New()
	memberSvc := &mockMemberService{}
	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/not-a-uuid/members", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_uuid", http.StatusBadRequest)
}

func TestMemberHandler_ListProjectMembers_AccessDenied(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	memberSvc := &mockMemberService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) ([]*dto.MemberResponse, error) {
			return nil, service.ErrAccessDenied
		},
	}

	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/members", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "access_denied", http.StatusForbidden)
}

func TestMemberHandler_GetMember_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	targetUserUUID := uuid.New()
	now := time.Now()

	memberSvc := &mockMemberService{
		getFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, tUUID uuid.UUID) (*dto.MemberResponse, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, targetUserUUID, tUUID)
			return &dto.MemberResponse{
				Username:    "member_user",
				UserUUID:    targetUserUUID.String(),
				ProjectUUID: projectUUID.String(),
				RoleID:      3,
				RoleName:    "member",
				JoinedAt:    now,
			}, nil
		},
	}

	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/member?userUUID="+targetUserUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var member dto.MemberResponse
	json.Unmarshal(w.Body.Bytes(), &member)
	assert.Equal(t, "member_user", member.Username)
	assert.Equal(t, "member", member.RoleName)
}

func TestMemberHandler_GetMember_MissingUserUUID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	memberSvc := &mockMemberService{}
	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/member", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "missing_user_uuid", http.StatusBadRequest)
}

func TestMemberHandler_GetMember_InvalidUserUUID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	memberSvc := &mockMemberService{}
	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/member?userUUID=not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_user_uuid", http.StatusBadRequest)
}

func TestMemberHandler_GetMember_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	targetUserUUID := uuid.New()
	memberSvc := &mockMemberService{
		getFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, tUUID uuid.UUID) (*dto.MemberResponse, error) {
			return nil, service.ErrMemberNotFound
		},
	}

	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/member?userUUID="+targetUserUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "member_not_found", http.StatusNotFound)
}

func TestMemberHandler_AddMember_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	targetUserUUID := uuid.New()
	now := time.Now()

	memberSvc := &mockMemberService{
		addFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, req dto.CreateMemberRequest) (*model.ProjectMember, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, targetUserUUID.String(), req.UserUUID)
			assert.Equal(t, 3, req.RoleID)
			return &model.ProjectMember{
				ProjectUUID: projectUUID,
				UserUUID:    targetUserUUID,
				RoleID:      3,
				JoinedAt:    now,
			}, nil
		},
	}

	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"user_uuid": "` + targetUserUUID.String() + `", "role_id": 3}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/members", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var member model.ProjectMember
	json.Unmarshal(w.Body.Bytes(), &member)
	assert.Equal(t, targetUserUUID, member.UserUUID)
	assert.Equal(t, 3, member.RoleID)
}

func TestMemberHandler_AddMember_InvalidUserUUID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	memberSvc := &mockMemberService{}
	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"user_uuid": "not-a-uuid", "role_id": 3}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/members", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "validation_error", http.StatusBadRequest)
}

func TestMemberHandler_AddMember_InvalidRoleID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	targetUserUUID := uuid.New()
	memberSvc := &mockMemberService{}
	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"user_uuid": "` + targetUserUUID.String() + `", "role_id": 0}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/members", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_request", http.StatusBadRequest)
}

func TestMemberHandler_AddMember_AlreadyExists(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	targetUserUUID := uuid.New()
	memberSvc := &mockMemberService{
		addFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, req dto.CreateMemberRequest) (*model.ProjectMember, error) {
			return nil, service.ErrMemberAlreadyExists
		},
	}

	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"user_uuid": "` + targetUserUUID.String() + `", "role_id": 3}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/members", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "member_already_exists", http.StatusConflict)
}

func TestMemberHandler_UpdateMemberRole_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	targetUserUUID := uuid.New()
	now := time.Now()

	memberSvc := &mockMemberService{
		updateRoleFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, tUUID uuid.UUID, req dto.UpdateMemberRequest) (*model.ProjectMember, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, targetUserUUID, tUUID)
			assert.Equal(t, 2, req.RoleID)
			return &model.ProjectMember{
				ProjectUUID: projectUUID,
				UserUUID:    targetUserUUID,
				RoleID:      2,
				JoinedAt:    now,
			}, nil
		},
	}

	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"role_id": 2}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/member?userUUID="+targetUserUUID.String(), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var member model.ProjectMember
	json.Unmarshal(w.Body.Bytes(), &member)
	assert.Equal(t, 2, member.RoleID)
}

func TestMemberHandler_UpdateMemberRole_InvalidRoleID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	targetUserUUID := uuid.New()
	memberSvc := &mockMemberService{}
	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"role_id": 0}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/member?userUUID="+targetUserUUID.String(), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_request", http.StatusBadRequest)
}

func TestMemberHandler_UpdateMemberRole_CannotRemoveLastOwner(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	targetUserUUID := uuid.New()
	memberSvc := &mockMemberService{
		updateRoleFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, tUUID uuid.UUID, req dto.UpdateMemberRequest) (*model.ProjectMember, error) {
			return nil, service.ErrCannotRemoveLastOwner
		},
	}

	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"role_id": 3}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/member?userUUID="+targetUserUUID.String(), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "cannot_remove_last_owner", http.StatusConflict)
}

func TestMemberHandler_RemoveMember_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	targetUserUUID := uuid.New()
	called := false

	memberSvc := &mockMemberService{
		removeFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, tUUID uuid.UUID) error {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, targetUserUUID, tUUID)
			called = true
			return nil
		},
	}

	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/member?userUUID="+targetUserUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	assert.True(t, called)
}

func TestMemberHandler_RemoveMember_CannotRemoveSelf(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	memberSvc := &mockMemberService{
		removeFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, tUUID uuid.UUID) error {
			return service.ErrCannotRemoveSelf
		},
	}

	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/member?userUUID="+userUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "cannot_remove_self", http.StatusForbidden)
}

func TestMemberHandler_RemoveMember_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	targetUserUUID := uuid.New()
	memberSvc := &mockMemberService{
		removeFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, tUUID uuid.UUID) error {
			return service.ErrMemberNotFound
		},
	}

	r := setupMemberRouter(t, memberSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/member?userUUID="+targetUserUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "member_not_found", http.StatusNotFound)
}

func TestMemberHandler_Unauthorized_AllEndpoints(t *testing.T) {
	memberSvc := &mockMemberService{}
	r := setupMemberRouter(t, memberSvc, "test-secret")

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/projects/" + uuid.New().String() + "/members", ""},
		{"POST", "/projects/" + uuid.New().String() + "/members", `{"user_uuid":"` + uuid.New().String() + `","role_id":3}`},
		{"GET", "/projects/" + uuid.New().String() + "/member?userUUID=" + uuid.New().String(), ""},
		{"PATCH", "/projects/" + uuid.New().String() + "/member?userUUID=" + uuid.New().String(), `{"role_id":2}`},
		{"DELETE", "/projects/" + uuid.New().String() + "/member?userUUID=" + uuid.New().String(), ""},
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
