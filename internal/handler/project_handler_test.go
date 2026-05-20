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
	dto "github.com/vega-trello/trello-back/internal/dto/project"
	"github.com/vega-trello/trello-back/internal/model"
	"github.com/vega-trello/trello-back/internal/service"
)

type mockProjectService struct {
	getUserFunc func(ctx context.Context, userUUID uuid.UUID) ([]*model.Project, error)
	createFunc  func(ctx context.Context, creatorUUID uuid.UUID, title string, description *string) (*model.Project, error)
	getFunc     func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) (*model.Project, error)
	updateFunc  func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, title *string, description *string) (*model.Project, error)
	deleteFunc  func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) error
}

func (m *mockProjectService) GetUserProjects(ctx context.Context, userUUID uuid.UUID) ([]*model.Project, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, userUUID)
	}
	return nil, nil
}

func (m *mockProjectService) CreateProject(ctx context.Context, creatorUUID uuid.UUID, title string, description *string) (*model.Project, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, creatorUUID, title, description)
	}
	return nil, nil
}

func (m *mockProjectService) GetProject(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) (*model.Project, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, projectUUID, userUUID)
	}
	return nil, nil
}

func (m *mockProjectService) UpdateProject(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, title *string, description *string) (*model.Project, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, projectUUID, userUUID, title, description)
	}
	return nil, nil
}

func (m *mockProjectService) DeleteProject(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, projectUUID, userUUID)
	}
	return nil
}

func setupProjectRouter(t *testing.T, projSvc *mockProjectService, jwtSecret string) *gin.Engine {
	t.Helper()

	h := NewProjectHandler(projSvc)

	return SetupTestRouterWithAuth(t, jwtSecret, func(rg *gin.RouterGroup) {
		rg.GET("/projects", h.ListProjects)
		rg.POST("/projects", h.CreateProject)
		rg.GET("/projects/:projectUUID", h.GetProject)
		rg.PATCH("/projects/:projectUUID", h.UpdateProject)
		rg.DELETE("/projects/:projectUUID", h.DeleteProject)
	})
}

func TestProjectHandler_ListProjects_Success(t *testing.T) {
	userUUID := uuid.New()
	projUUID := uuid.New()
	desc := "Test project"

	projSvc := &mockProjectService{
		getUserFunc: func(ctx context.Context, u uuid.UUID) ([]*model.Project, error) {
			assert.Equal(t, userUUID, u)
			return []*model.Project{
				{
					UUID:        projUUID,
					Title:       "Test Project",
					Description: &desc,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				},
			}, nil
		},
	}
	r := setupProjectRouter(t, projSvc, "test-secret")

	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var projects []*dto.ProjectResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &projects))
	require.Len(t, projects, 1)
	assert.Equal(t, "Test Project", projects[0].Title)
	assert.Equal(t, "Test project", *projects[0].Description)
}

func TestProjectHandler_ListProjects_Unauthorized(t *testing.T) {
	projSvc := &mockProjectService{}
	r := setupProjectRouter(t, projSvc, "test-secret")

	req := httptest.NewRequest("GET", "/projects", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "missing_token", http.StatusUnauthorized)
}

func TestProjectHandler_ListProjects_ServiceError(t *testing.T) {
	userUUID := uuid.New()
	projSvc := &mockProjectService{
		getUserFunc: func(ctx context.Context, u uuid.UUID) ([]*model.Project, error) {
			return nil, service.ErrAccessDenied
		},
	}

	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "access_denied", http.StatusForbidden)
}

func TestProjectHandler_CreateProject_Success(t *testing.T) {
	userUUID := uuid.New()
	projUUID := uuid.New()
	desc := "New project description"

	projSvc := &mockProjectService{
		createFunc: func(ctx context.Context, creator uuid.UUID, title string, description *string) (*model.Project, error) {
			assert.Equal(t, userUUID, creator)
			assert.Equal(t, "New Project", title)
			assert.NotNil(t, description)
			assert.Equal(t, desc, *description)
			return &model.Project{
				UUID:        projUUID,
				Title:       title,
				Description: description,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"title":"New Project","description":"New project description"}`)
	req := httptest.NewRequest("POST", "/projects", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var proj dto.ProjectResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &proj))
	assert.Equal(t, "New Project", proj.Title)
	assert.Equal(t, "New project description", *proj.Description)
}

func TestProjectHandler_CreateProject_WithoutDescription(t *testing.T) {
	userUUID := uuid.New()
	projUUID := uuid.New()

	projSvc := &mockProjectService{
		createFunc: func(ctx context.Context, creator uuid.UUID, title string, description *string) (*model.Project, error) {
			assert.Nil(t, description)
			return &model.Project{
				UUID:        projUUID,
				Title:       title,
				Description: nil,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"title":"Minimal Project"}`)
	req := httptest.NewRequest("POST", "/projects", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var proj dto.ProjectResponse
	json.Unmarshal(w.Body.Bytes(), &proj)
	assert.Nil(t, proj.Description)
}

func TestProjectHandler_CreateProject_InvalidJSON(t *testing.T) {
	userUUID := uuid.New()
	projSvc := &mockProjectService{}
	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"title":}`)
	req := httptest.NewRequest("POST", "/projects", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_request", http.StatusBadRequest)
}

func TestProjectHandler_CreateProject_ValidationFailed(t *testing.T) {
	userUUID := uuid.New()
	projSvc := &mockProjectService{}
	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"title":"","description":"ok"}`)
	req := httptest.NewRequest("POST", "/projects", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_request", http.StatusBadRequest)
}

func TestProjectHandler_CreateProject_ServiceError(t *testing.T) {
	userUUID := uuid.New()
	projSvc := &mockProjectService{
		createFunc: func(ctx context.Context, creator uuid.UUID, title string, description *string) (*model.Project, error) {
			return nil, service.ErrProjectTitleTaken
		},
	}

	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"title":"Taken Title","description":"desc"}`)
	req := httptest.NewRequest("POST", "/projects", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "project_title_taken", http.StatusConflict)
}

func TestProjectHandler_GetProject_Success(t *testing.T) {
	userUUID := uuid.New()
	projUUID := uuid.New()
	desc := "Project details"

	projSvc := &mockProjectService{
		getFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) (*model.Project, error) {
			assert.Equal(t, projUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			return &model.Project{
				UUID:        projUUID,
				Title:       "My Project",
				Description: &desc,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var proj dto.ProjectResponse
	json.Unmarshal(w.Body.Bytes(), &proj)
	assert.Equal(t, projUUID.String(), proj.UUID)
	assert.Equal(t, "My Project", proj.Title)
}

func TestProjectHandler_GetProject_InvalidUUID(t *testing.T) {
	userUUID := uuid.New()
	projSvc := &mockProjectService{}
	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_uuid", http.StatusBadRequest)
}

func TestProjectHandler_GetProject_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projUUID := uuid.New()
	projSvc := &mockProjectService{
		getFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) (*model.Project, error) {
			return nil, service.ErrProjectNotFound
		},
	}

	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "project_not_found", http.StatusNotFound)
}

func TestProjectHandler_UpdateProject_Success_FullUpdate(t *testing.T) {
	userUUID := uuid.New()
	projUUID := uuid.New()
	newTitle := "Updated Title"
	newDesc := "Updated description"

	projSvc := &mockProjectService{
		updateFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, title *string, description *string) (*model.Project, error) {
			assert.Equal(t, projUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			assert.NotNil(t, title)
			assert.Equal(t, newTitle, *title)
			assert.NotNil(t, description)
			assert.Equal(t, newDesc, *description)
			return &model.Project{
				UUID:        projUUID,
				Title:       newTitle,
				Description: description,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"title":"Updated Title","description":"Updated description"}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projUUID.String(), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var proj dto.ProjectResponse
	json.Unmarshal(w.Body.Bytes(), &proj)
	assert.Equal(t, newTitle, proj.Title)
	assert.Equal(t, newDesc, *proj.Description)
}

func TestProjectHandler_UpdateProject_Success_PartialUpdate_TitleOnly(t *testing.T) {
	userUUID := uuid.New()
	projUUID := uuid.New()
	newTitle := "Only Title Changed"

	projSvc := &mockProjectService{
		updateFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, title *string, description *string) (*model.Project, error) {
			assert.NotNil(t, title)
			assert.Nil(t, description)
			assert.Equal(t, newTitle, *title)
			return &model.Project{
				UUID:        projUUID,
				Title:       newTitle,
				Description: nil,
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"title":"Only Title Changed"}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projUUID.String(), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var proj dto.ProjectResponse
	json.Unmarshal(w.Body.Bytes(), &proj)
	assert.Equal(t, newTitle, proj.Title)
}

func TestProjectHandler_UpdateProject_Success_PartialUpdate_DescriptionOnly(t *testing.T) {
	userUUID := uuid.New()
	projUUID := uuid.New()
	newDesc := "Only description changed"

	projSvc := &mockProjectService{
		updateFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, title *string, description *string) (*model.Project, error) {
			assert.Nil(t, title)
			assert.NotNil(t, description)
			assert.Equal(t, newDesc, *description)
			return &model.Project{
				UUID:        projUUID,
				Description: description,
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"description":"Only description changed"}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projUUID.String(), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var proj dto.ProjectResponse
	json.Unmarshal(w.Body.Bytes(), &proj)
	assert.Equal(t, newDesc, *proj.Description)
}

func TestProjectHandler_UpdateProject_ValidationFailed(t *testing.T) {
	userUUID := uuid.New()
	projUUID := uuid.New()
	projSvc := &mockProjectService{}
	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"title":"` + string(make([]byte, 129)) + `"}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projUUID.String(), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_request", http.StatusBadRequest)
}

func TestProjectHandler_DeleteProject_Success(t *testing.T) {
	userUUID := uuid.New()
	projUUID := uuid.New()
	called := false

	projSvc := &mockProjectService{
		deleteFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) error {
			assert.Equal(t, projUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			called = true
			return nil
		},
	}

	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	assert.True(t, called)
}

func TestProjectHandler_DeleteProject_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projUUID := uuid.New()
	projSvc := &mockProjectService{
		deleteFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) error {
			return service.ErrProjectNotFound
		},
	}

	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "project_not_found", http.StatusNotFound)
}

func TestProjectHandler_DeleteProject_AccessDenied(t *testing.T) {
	userUUID := uuid.New()
	projUUID := uuid.New()
	projSvc := &mockProjectService{
		deleteFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) error {
			return service.ErrAccessDenied
		},
	}

	r := setupProjectRouter(t, projSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projUUID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "access_denied", http.StatusForbidden)
}

func TestProjectHandler_Unauthorized_AllEndpoints(t *testing.T) {
	projSvc := &mockProjectService{}
	r := setupProjectRouter(t, projSvc, "test-secret")

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/projects", ""},
		{"POST", "/projects", `{"title":"test"}`},
		{"GET", "/projects/" + uuid.New().String(), ""},
		{"PATCH", "/projects/" + uuid.New().String(), `{"title":"new"}`},
		{"DELETE", "/projects/" + uuid.New().String(), ""},
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
