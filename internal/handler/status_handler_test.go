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
	dto "github.com/vega-trello/trello-back/internal/dto/status"
	"github.com/vega-trello/trello-back/internal/model"
	"github.com/vega-trello/trello-back/internal/service"
)

type mockStatusService struct {
	createFunc func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto.CreateStatusRequest) (*model.ProjectStatus, error)
	listFunc   func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.ProjectStatus, error)
	getFunc    func(ctx context.Context, projectUUID uuid.UUID, statusID int, userUUID uuid.UUID) (*model.ProjectStatus, error)
	updateFunc func(ctx context.Context, projectUUID uuid.UUID, statusID int, userUUID uuid.UUID, req dto.UpdateStatusRequest) (*model.ProjectStatus, error)
	deleteFunc func(ctx context.Context, projectUUID uuid.UUID, statusID int, userUUID uuid.UUID) error
}

func (m *mockStatusService) CreateStatus(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto.CreateStatusRequest) (*model.ProjectStatus, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, projectUUID, userUUID, req)
	}
	return nil, nil
}
func (m *mockStatusService) GetProjectStatuses(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.ProjectStatus, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, projectUUID, userUUID)
	}
	return nil, nil
}
func (m *mockStatusService) GetStatus(ctx context.Context, projectUUID uuid.UUID, statusID int, userUUID uuid.UUID) (*model.ProjectStatus, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, projectUUID, statusID, userUUID)
	}
	return nil, nil
}
func (m *mockStatusService) UpdateStatus(ctx context.Context, projectUUID uuid.UUID, statusID int, userUUID uuid.UUID, req dto.UpdateStatusRequest) (*model.ProjectStatus, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, projectUUID, statusID, userUUID, req)
	}
	return nil, nil
}
func (m *mockStatusService) DeleteStatus(ctx context.Context, projectUUID uuid.UUID, statusID int, userUUID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, projectUUID, statusID, userUUID)
	}
	return nil
}

func setupStatusRouter(t *testing.T, statusSvc *mockStatusService, jwtSecret string) *gin.Engine {
	t.Helper()

	h := NewStatusHandler(statusSvc)

	return SetupTestRouterWithAuth(t, jwtSecret, func(rg *gin.RouterGroup) {
		rg.GET("/projects/:projectUUID/statuses", h.ListProjectStatuses)
		rg.POST("/projects/:projectUUID/statuses", h.CreateStatus)
		rg.GET("/projects/:projectUUID/statuses/:statusID", h.GetStatus)
		rg.PATCH("/projects/:projectUUID/statuses/:statusID", h.UpdateStatus)
		rg.DELETE("/projects/:projectUUID/statuses/:statusID", h.DeleteStatus)
	})
}

func TestStatusHandler_ListProjectStatuses_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	now := time.Now()

	statusSvc := &mockStatusService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) ([]*model.ProjectStatus, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			return []*model.ProjectStatus{
				{ID: 1, ProjectUUID: projectUUID, Name: "To Do", CreatedAt: now},
				{ID: 2, ProjectUUID: projectUUID, Name: "In Progress", CreatedAt: now},
			}, nil
		},
	}

	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/statuses", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var statuses []*model.ProjectStatus
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &statuses))
	require.Len(t, statuses, 2)
	assert.Equal(t, "To Do", statuses[0].Name)
	assert.Equal(t, "In Progress", statuses[1].Name)
}

func TestStatusHandler_ListProjectStatuses_InvalidUUID(t *testing.T) {
	userUUID := uuid.New()
	statusSvc := &mockStatusService{}
	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/not-a-uuid/statuses", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_uuid", http.StatusBadRequest)
}

func TestStatusHandler_ListProjectStatuses_AccessDenied(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	statusSvc := &mockStatusService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) ([]*model.ProjectStatus, error) {
			return nil, service.ErrAccessDenied
		},
	}

	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/statuses", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "access_denied", http.StatusForbidden)
}

func TestStatusHandler_CreateStatus_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	statusID := 3
	now := time.Now()

	statusSvc := &mockStatusService{
		createFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, req dto.CreateStatusRequest) (*model.ProjectStatus, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, "Done", req.Name)
			return &model.ProjectStatus{
				ID:          statusID,
				ProjectUUID: projectUUID,
				Name:        req.Name,
				CreatedAt:   now,
			}, nil
		},
	}

	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Done"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/statuses", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var status model.ProjectStatus
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &status))
	assert.Equal(t, "Done", status.Name)
	assert.Equal(t, statusID, status.ID)
}

func TestStatusHandler_CreateStatus_InvalidName(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	statusSvc := &mockStatusService{}
	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":""}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/statuses", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_request", http.StatusBadRequest)
}

func TestStatusHandler_CreateStatus_AlreadyExists(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	statusSvc := &mockStatusService{
		createFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, req dto.CreateStatusRequest) (*model.ProjectStatus, error) {
			return nil, service.ErrStatusAlreadyExists
		},
	}

	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Done"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/statuses", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "status_already_exists", http.StatusConflict)
}

func TestStatusHandler_GetStatus_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	statusID := 3
	now := time.Now()

	statusSvc := &mockStatusService{
		getFunc: func(ctx context.Context, pUUID uuid.UUID, sID int, uUUID uuid.UUID) (*model.ProjectStatus, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, statusID, sID)
			assert.Equal(t, userUUID, uUUID)
			return &model.ProjectStatus{
				ID:          statusID,
				ProjectUUID: projectUUID,
				Name:        "Done",
				CreatedAt:   now,
			}, nil
		},
	}

	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	// 🔹 statusID в path!
	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/statuses/"+strconv.Itoa(statusID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var status model.ProjectStatus
	json.Unmarshal(w.Body.Bytes(), &status)
	assert.Equal(t, "Done", status.Name)
	assert.Equal(t, statusID, status.ID)
}

func TestStatusHandler_GetStatus_InvalidStatusID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	statusSvc := &mockStatusService{}
	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	// 🔹 statusID не число
	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/statuses/abc", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_status_id", http.StatusBadRequest)
}

func TestStatusHandler_GetStatus_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	statusID := 999
	statusSvc := &mockStatusService{
		getFunc: func(ctx context.Context, pUUID uuid.UUID, sID int, uUUID uuid.UUID) (*model.ProjectStatus, error) {
			return nil, service.ErrStatusNotFound
		},
	}

	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/statuses/"+strconv.Itoa(statusID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "status_not_found", http.StatusNotFound)
}

func TestStatusHandler_UpdateStatus_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	statusID := 3
	now := time.Now()

	statusSvc := &mockStatusService{
		updateFunc: func(ctx context.Context, pUUID uuid.UUID, sID int, uUUID uuid.UUID, req dto.UpdateStatusRequest) (*model.ProjectStatus, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, statusID, sID)
			assert.Equal(t, "Completed", req.Name)
			return &model.ProjectStatus{
				ID:          statusID,
				ProjectUUID: projectUUID,
				Name:        req.Name,
				CreatedAt:   now,
			}, nil
		},
	}

	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Completed"}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/statuses/"+strconv.Itoa(statusID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var status model.ProjectStatus
	json.Unmarshal(w.Body.Bytes(), &status)
	assert.Equal(t, "Completed", status.Name)
}

func TestStatusHandler_UpdateStatus_InvalidName(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	statusID := 3
	statusSvc := &mockStatusService{}
	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":""}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/statuses/"+strconv.Itoa(statusID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_request", http.StatusBadRequest)
}

func TestStatusHandler_UpdateStatus_AlreadyExists(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	statusID := 3
	statusSvc := &mockStatusService{
		updateFunc: func(ctx context.Context, pUUID uuid.UUID, sID int, uUUID uuid.UUID, req dto.UpdateStatusRequest) (*model.ProjectStatus, error) {
			return nil, service.ErrStatusAlreadyExists
		},
	}

	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Done"}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/statuses/"+strconv.Itoa(statusID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "status_already_exists", http.StatusConflict)
}

func TestStatusHandler_DeleteStatus_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	statusID := 3
	called := false

	statusSvc := &mockStatusService{
		deleteFunc: func(ctx context.Context, pUUID uuid.UUID, sID int, uUUID uuid.UUID) error {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, statusID, sID)
			assert.Equal(t, userUUID, uUUID)
			called = true
			return nil
		},
	}

	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/statuses/"+strconv.Itoa(statusID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	assert.True(t, called)
}

func TestStatusHandler_DeleteStatus_HasActiveTasks(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	statusID := 3
	statusSvc := &mockStatusService{
		deleteFunc: func(ctx context.Context, pUUID uuid.UUID, sID int, uUUID uuid.UUID) error {
			return service.ErrStatusHasActiveTasks
		},
	}

	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/statuses/"+strconv.Itoa(statusID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "status_has_active_tasks", http.StatusConflict)
}

func TestStatusHandler_DeleteStatus_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	statusID := 999
	statusSvc := &mockStatusService{
		deleteFunc: func(ctx context.Context, pUUID uuid.UUID, sID int, uUUID uuid.UUID) error {
			return service.ErrStatusNotFound
		},
	}

	r := setupStatusRouter(t, statusSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/statuses/"+strconv.Itoa(statusID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "status_not_found", http.StatusNotFound)
}

func TestStatusHandler_Unauthorized_AllEndpoints(t *testing.T) {
	statusSvc := &mockStatusService{}
	r := setupStatusRouter(t, statusSvc, "test-secret")

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/projects/" + uuid.New().String() + "/statuses", ""},
		{"POST", "/projects/" + uuid.New().String() + "/statuses", `{"name":"Test"}`},
		{"GET", "/projects/" + uuid.New().String() + "/statuses/1", ""},
		{"PATCH", "/projects/" + uuid.New().String() + "/statuses/1", `{"name":"New"}`},
		{"DELETE", "/projects/" + uuid.New().String() + "/statuses/1", ""},
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
