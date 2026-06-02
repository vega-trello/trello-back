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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dto "github.com/vega-trello/trello-back/internal/dto/assignee"
	"github.com/vega-trello/trello-back/internal/service"
)

type mockAssigneeService struct {
	listFunc   func(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) ([]*dto.AssigneeResponse, error)
	assignFunc func(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, req dto.CreateAssigneeRequest) error
	removeFunc func(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, assigneeUUID uuid.UUID) error
}

func (m *mockAssigneeService) GetTaskAssignees(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) ([]*dto.AssigneeResponse, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, projectUUID, taskID, userUUID)
	}
	return nil, nil
}

func (m *mockAssigneeService) AssignUserToTask(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, req dto.CreateAssigneeRequest) error {
	if m.assignFunc != nil {
		return m.assignFunc(ctx, projectUUID, taskID, userUUID, req)
	}
	return nil
}

func (m *mockAssigneeService) RemoveAssignee(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, assigneeUUID uuid.UUID) error {
	if m.removeFunc != nil {
		return m.removeFunc(ctx, projectUUID, taskID, userUUID, assigneeUUID)
	}
	return nil
}

func setupAssigneeRouter(t *testing.T, assigneeSvc *mockAssigneeService, jwtSecret string) *gin.Engine {
	t.Helper()

	h := NewAssigneeHandler(assigneeSvc)

	return SetupTestRouterWithAuth(t, jwtSecret, func(rg *gin.RouterGroup) {
		rg.GET("/projects/:projectUUID/assignees", h.ListTaskAssignees)
		rg.POST("/projects/:projectUUID/assignees", h.AddAssignee)
		rg.DELETE("/projects/:projectUUID/assignee", h.RemoveAssignee)
	})
}

func TestAssigneeHandler_ListTaskAssignees_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	assigneeUUID := uuid.New()
	now := time.Now()

	assigneeSvc := &mockAssigneeService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID) ([]*dto.AssigneeResponse, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, userUUID, uUUID)
			return []*dto.AssigneeResponse{
				{
					TaskID:     taskID,
					UserUUID:   assigneeUUID.String(),
					AssignedAt: now,
					User: &dto.UserInfo{
						Username: "assigned_user",
						UUID:     assigneeUUID.String(),
					},
				},
			}, nil
		},
	}

	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/assignees?taskID="+strconv.Itoa(taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []dto.AssigneeResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	require.Len(t, resp, 1)
	assert.Equal(t, "assigned_user", resp[0].User.Username)
	assert.Equal(t, taskID, resp[0].TaskID)
	assert.Equal(t, assigneeUUID.String(), resp[0].UserUUID)
}

func TestAssigneeHandler_ListTaskAssignees_Empty_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105

	assigneeSvc := &mockAssigneeService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID) ([]*dto.AssigneeResponse, error) {
			return []*dto.AssigneeResponse{}, nil
		},
	}

	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/assignees?taskID="+strconv.Itoa(taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []dto.AssigneeResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp)
	assert.Equal(t, "[]", w.Body.String())
}

func TestAssigneeHandler_ListTaskAssignees_MissingTaskID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	assigneeSvc := &mockAssigneeService{}
	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/assignees", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "missing_task_id", http.StatusBadRequest)
}

func TestAssigneeHandler_ListTaskAssignees_InvalidTaskID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	assigneeSvc := &mockAssigneeService{}
	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/assignees?taskID=abc", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_task_id", http.StatusBadRequest)
}

func TestAssigneeHandler_ListTaskAssignees_AccessDenied(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	assigneeSvc := &mockAssigneeService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID) ([]*dto.AssigneeResponse, error) {
			return nil, service.ErrAccessDenied
		},
	}

	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/assignees?taskID="+strconv.Itoa(taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "access_denied", http.StatusForbidden)
}

func TestAssigneeHandler_AddAssignee_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	assigneeUUID := uuid.New()

	assigneeSvc := &mockAssigneeService{
		assignFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID, req dto.CreateAssigneeRequest) error {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, assigneeUUID.String(), req.UserUUID)
			return nil
		},
	}

	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"user_uuid": "` + assigneeUUID.String() + `"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/assignees?taskID="+strconv.Itoa(taskID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAssigneeHandler_AddAssignee_MissingTaskID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	assigneeUUID := uuid.New()
	assigneeSvc := &mockAssigneeService{}
	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"user_uuid": "` + assigneeUUID.String() + `"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/assignees", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "missing_task_id", http.StatusBadRequest)
}

func TestAssigneeHandler_AddAssignee_InvalidUserUUID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105

	assigneeSvc := &mockAssigneeService{
		assignFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID, req dto.CreateAssigneeRequest) error {
			if _, err := uuid.Parse(req.UserUUID); err != nil {
				return service.ErrInvalidUserUUID
			}
			return nil
		},
	}

	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"user_uuid": "not-a-uuid"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/assignees?taskID="+strconv.Itoa(taskID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_user_uuid", http.StatusBadRequest)
}

func TestAssigneeHandler_AddAssignee_AlreadyAssigned(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	assigneeUUID := uuid.New()
	assigneeSvc := &mockAssigneeService{
		assignFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID, req dto.CreateAssigneeRequest) error {
			return service.ErrAlreadyAssigned
		},
	}

	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"user_uuid": "` + assigneeUUID.String() + `"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/assignees?taskID="+strconv.Itoa(taskID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "already_assigned", http.StatusConflict)
}

func TestAssigneeHandler_AddAssignee_UserNotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	assigneeUUID := uuid.New()
	assigneeSvc := &mockAssigneeService{
		assignFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID, req dto.CreateAssigneeRequest) error {
			return service.ErrUserNotFound
		},
	}

	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"user_uuid": "` + assigneeUUID.String() + `"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/assignees?taskID="+strconv.Itoa(taskID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "user_not_found", http.StatusNotFound)
}

func TestAssigneeHandler_RemoveAssignee_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	assigneeUUID := uuid.New()
	called := false

	assigneeSvc := &mockAssigneeService{
		removeFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID, aUUID uuid.UUID) error {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, assigneeUUID, aUUID)
			called = true
			return nil
		},
	}

	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/assignee?taskID="+strconv.Itoa(taskID)+"&userUUID="+assigneeUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	assert.True(t, called)
}

func TestAssigneeHandler_RemoveAssignee_MissingTaskID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	assigneeUUID := uuid.New()
	assigneeSvc := &mockAssigneeService{}
	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/assignee?userUUID="+assigneeUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "missing_task_id", http.StatusBadRequest)
}

func TestAssigneeHandler_RemoveAssignee_MissingUserUUID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	assigneeSvc := &mockAssigneeService{}
	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/assignee?taskID="+strconv.Itoa(taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "missing_user_uuid", http.StatusBadRequest)
}

func TestAssigneeHandler_RemoveAssignee_InvalidUserUUID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	assigneeSvc := &mockAssigneeService{}
	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/assignee?taskID="+strconv.Itoa(taskID)+"&userUUID=not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_user_uuid", http.StatusBadRequest)
}

func TestAssigneeHandler_RemoveAssignee_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	assigneeUUID := uuid.New()
	assigneeSvc := &mockAssigneeService{
		removeFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID, aUUID uuid.UUID) error {
			return service.ErrAssigneeNotFound
		},
	}

	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/assignee?taskID="+strconv.Itoa(taskID)+"&userUUID="+assigneeUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "assignee_not_found", http.StatusNotFound)
}

func TestAssigneeHandler_Unauthorized_AllEndpoints(t *testing.T) {
	assigneeSvc := &mockAssigneeService{}
	r := setupAssigneeRouter(t, assigneeSvc, "test-secret")

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/projects/" + uuid.New().String() + "/assignees?taskID=1", ""},
		{"POST", "/projects/" + uuid.New().String() + "/assignees?taskID=1", `{"user_uuid":"` + uuid.New().String() + `"}`},
		{"DELETE", "/projects/" + uuid.New().String() + "/assignee?taskID=1&userUUID=" + uuid.New().String(), ""},
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
