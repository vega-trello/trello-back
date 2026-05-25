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
	dto "github.com/vega-trello/trello-back/internal/dto/tag"
	"github.com/vega-trello/trello-back/internal/model"
	"github.com/vega-trello/trello-back/internal/service"
)

type mockTagService struct {
	listProjectFunc    func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Tag, error)
	listTaskFunc       func(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) ([]*model.Tag, error)
	createFunc         func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto.CreateTagRequest) (*model.Tag, error)
	updateFunc         func(ctx context.Context, tagID int, userUUID uuid.UUID, req dto.UpdateTagRequest) (*model.Tag, error)
	deleteFunc         func(ctx context.Context, tagID int, userUUID uuid.UUID) error
	addToTaskFunc      func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, taskID int, tagID int) error
	removeFromTaskFunc func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, taskID int, tagID int) error
}

func (m *mockTagService) GetProjectTags(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Tag, error) {
	if m.listProjectFunc != nil {
		return m.listProjectFunc(ctx, projectUUID, userUUID)
	}
	return nil, nil
}
func (m *mockTagService) GetTaskTags(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) ([]*model.Tag, error) {
	if m.listTaskFunc != nil {
		return m.listTaskFunc(ctx, projectUUID, taskID, userUUID)
	}
	return nil, nil
}
func (m *mockTagService) CreateTag(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto.CreateTagRequest) (*model.Tag, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, projectUUID, userUUID, req)
	}
	return nil, nil
}
func (m *mockTagService) UpdateTag(ctx context.Context, tagID int, userUUID uuid.UUID, req dto.UpdateTagRequest) (*model.Tag, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, tagID, userUUID, req)
	}
	return nil, nil
}
func (m *mockTagService) DeleteTag(ctx context.Context, tagID int, userUUID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, tagID, userUUID)
	}
	return nil
}
func (m *mockTagService) AddTagToTask(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, taskID int, tagID int) error {
	if m.addToTaskFunc != nil {
		return m.addToTaskFunc(ctx, projectUUID, userUUID, taskID, tagID)
	}
	return nil
}
func (m *mockTagService) RemoveTagFromTask(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, taskID int, tagID int) error {
	if m.removeFromTaskFunc != nil {
		return m.removeFromTaskFunc(ctx, projectUUID, userUUID, taskID, tagID)
	}
	return nil
}

func setupTagRouter(t *testing.T, tagSvc *mockTagService, jwtSecret string) *gin.Engine {
	t.Helper()

	h := NewTagHandler(tagSvc)

	return SetupTestRouterWithAuth(t, jwtSecret, func(rg *gin.RouterGroup) {
		rg.GET("/projects/:projectUUID/tag", h.ListProjectTags)
		rg.POST("/projects/:projectUUID/tag", h.CreateTag)
		rg.PATCH("/projects/:projectUUID/tag", h.UpdateTag)
		rg.DELETE("/projects/:projectUUID/tag", h.DeleteTag)

		rg.GET("/projects/:projectUUID/task/tags", h.ListTaskTags)
		rg.POST("/projects/:projectUUID/task/tags", h.AddTagToTask)
		rg.DELETE("/projects/:projectUUID/task/tags/:tagID", h.RemoveTagFromTask)
	})
}

func TestTagHandler_ListProjectTags_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagID := 15
	now := time.Now()

	tagSvc := &mockTagService{
		listProjectFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) ([]*model.Tag, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			return []*model.Tag{
				{
					ID:          tagID,
					ProjectUUID: projectUUID,
					Name:        "Bug",
					Color:       "#FF0000",
					CreatedAt:   now,
					UpdatedAt:   now,
				},
			}, nil
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/tag", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var tags []dto.TagResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tags))
	require.Len(t, tags, 1)
	assert.Equal(t, "Bug", tags[0].Name)
	assert.Equal(t, "#FF0000", tags[0].Color)
	assert.Equal(t, tagID, tags[0].ID)
}

func TestTagHandler_ListProjectTags_InvalidUUID(t *testing.T) {
	userUUID := uuid.New()
	tagSvc := &mockTagService{}
	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/not-a-uuid/tag", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_uuid", http.StatusBadRequest)
}

func TestTagHandler_ListProjectTags_AccessDenied(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagSvc := &mockTagService{
		listProjectFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) ([]*model.Tag, error) {
			return nil, service.ErrAccessDenied
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/tag", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "access_denied", http.StatusForbidden)
}

func TestTagHandler_CreateTag_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagID := 15
	now := time.Now()

	tagSvc := &mockTagService{
		createFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, req dto.CreateTagRequest) (*model.Tag, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, "Feature", req.Name)
			assert.Equal(t, "#00FF00", req.Color)
			return &model.Tag{
				ID:          tagID,
				ProjectUUID: projectUUID,
				Name:        req.Name,
				Color:       req.Color,
				CreatedAt:   now,
				UpdatedAt:   now,
			}, nil
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Feature","color":"#00FF00"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/tag", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var tag dto.TagResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tag))
	assert.Equal(t, "Feature", tag.Name)
	assert.Equal(t, "#00FF00", tag.Color)
}

func TestTagHandler_CreateTag_InvalidName(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagSvc := &mockTagService{}
	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"","color":"#FF0000"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/tag", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "validation_error", http.StatusBadRequest)
}

func TestTagHandler_CreateTag_InvalidColor(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagSvc := &mockTagService{}
	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Bug","color":"red"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/tag", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "validation_error", http.StatusBadRequest)
}

func TestTagHandler_CreateTag_AlreadyExists(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagSvc := &mockTagService{
		createFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, req dto.CreateTagRequest) (*model.Tag, error) {
			return nil, service.ErrTagAlreadyExists
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Bug","color":"#FF0000"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/tag", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "tag_already_exists", http.StatusConflict)
}

func TestTagHandler_UpdateTag_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagID := 15
	now := time.Now()

	tagSvc := &mockTagService{
		updateFunc: func(ctx context.Context, tID int, uUUID uuid.UUID, req dto.UpdateTagRequest) (*model.Tag, error) {
			assert.Equal(t, tagID, tID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, "Critical Bug", req.Name)
			assert.Equal(t, "#AA0000", req.Color)
			return &model.Tag{
				ID:          tagID,
				ProjectUUID: projectUUID,
				Name:        req.Name,
				Color:       req.Color,
				CreatedAt:   now,
				UpdatedAt:   now,
			}, nil
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Critical Bug","color":"#AA0000"}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/tag?tagID="+strconv.Itoa(tagID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var tag dto.TagResponse
	json.Unmarshal(w.Body.Bytes(), &tag)
	assert.Equal(t, "Critical Bug", tag.Name)
	assert.Equal(t, "#AA0000", tag.Color)
}

func TestTagHandler_UpdateTag_MissingTagID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagSvc := &mockTagService{}
	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"New","color":"#000000"}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/tag", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "missing_tag_id", http.StatusBadRequest)
}

func TestTagHandler_UpdateTag_InvalidTagID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagSvc := &mockTagService{}
	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"New","color":"#000000"}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/tag?tagID=abc", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_tag_id", http.StatusBadRequest)
}

func TestTagHandler_UpdateTag_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagID := 999
	tagSvc := &mockTagService{
		updateFunc: func(ctx context.Context, tID int, uUUID uuid.UUID, req dto.UpdateTagRequest) (*model.Tag, error) {
			return nil, service.ErrTagNotFound
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"New","color":"#000000"}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/tag?tagID="+strconv.Itoa(tagID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "tag_not_found", http.StatusNotFound)
}

func TestTagHandler_DeleteTag_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagID := 15
	called := false

	tagSvc := &mockTagService{
		deleteFunc: func(ctx context.Context, tID int, uUUID uuid.UUID) error {
			assert.Equal(t, tagID, tID)
			assert.Equal(t, userUUID, uUUID)
			called = true
			return nil
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/tag?tagID="+strconv.Itoa(tagID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	assert.True(t, called)
}

func TestTagHandler_DeleteTag_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagID := 999
	tagSvc := &mockTagService{
		deleteFunc: func(ctx context.Context, tID int, uUUID uuid.UUID) error {
			return service.ErrTagNotFound
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/tag?tagID="+strconv.Itoa(tagID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "tag_not_found", http.StatusNotFound)
}

func TestTagHandler_ListTaskTags_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	tagID := 15
	now := time.Now()

	tagSvc := &mockTagService{
		listTaskFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID) ([]*model.Tag, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, userUUID, uUUID)
			return []*model.Tag{
				{
					ID:          tagID,
					ProjectUUID: projectUUID,
					Name:        "Bug",
					Color:       "#FF0000",
					CreatedAt:   now,
					UpdatedAt:   now,
				},
			}, nil
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/task/tags?taskID="+strconv.Itoa(taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var tags []dto.TagResponse
	json.Unmarshal(w.Body.Bytes(), &tags)
	require.Len(t, tags, 1)
	assert.Equal(t, "Bug", tags[0].Name)
}

func TestTagHandler_ListTaskTags_MissingTaskID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagSvc := &mockTagService{}
	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/task/tags", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "missing_task_id", http.StatusBadRequest)
}

func TestTagHandler_ListTaskTags_TaskNotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 999
	tagSvc := &mockTagService{
		listTaskFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID) ([]*model.Tag, error) {
			return nil, service.ErrTaskNotFound
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/task/tags?taskID="+strconv.Itoa(taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "task_not_found", http.StatusNotFound)
}

func TestTagHandler_AddTagToTask_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	tagID := 15
	called := false

	tagSvc := &mockTagService{
		addToTaskFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, tID int, tgID int) error {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, tagID, tgID)
			called = true
			return nil
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/task/tags?taskID="+strconv.Itoa(taskID)+"&tagID="+strconv.Itoa(tagID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	assert.True(t, called)
}

func TestTagHandler_AddTagToTask_MissingTaskID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagID := 15
	tagSvc := &mockTagService{}
	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	// 🔹 Нет taskID в query
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/task/tags?tagID="+strconv.Itoa(tagID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "missing_task_id", http.StatusBadRequest)
}

func TestTagHandler_AddTagToTask_MissingTagID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	tagSvc := &mockTagService{}
	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/task/tags?taskID="+strconv.Itoa(taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "missing_tag_id", http.StatusBadRequest)
}

func TestTagHandler_AddTagToTask_InvalidTaskID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagSvc := &mockTagService{}
	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/task/tags?taskID=abc&tagID=15", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_task_id", http.StatusBadRequest)
}

func TestTagHandler_AddTagToTask_InvalidTagID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	tagSvc := &mockTagService{}
	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/task/tags?taskID="+strconv.Itoa(taskID)+"&tagID=abc", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_tag_id", http.StatusBadRequest)
}

func TestTagHandler_AddTagToTask_TagNotInProject(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	tagID := 15
	tagSvc := &mockTagService{
		addToTaskFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, tID int, tgID int) error {
			return service.ErrTagNotInProject
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/task/tags?taskID="+strconv.Itoa(taskID)+"&tagID="+strconv.Itoa(tagID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "tag_not_in_project", http.StatusBadRequest)
}

func TestTagHandler_AddTagToTask_AlreadyAttached(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	tagID := 15
	tagSvc := &mockTagService{
		addToTaskFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, tID int, tgID int) error {
			return service.ErrTagAlreadyAttached
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/task/tags?taskID="+strconv.Itoa(taskID)+"&tagID="+strconv.Itoa(tagID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "tag_already_attached", http.StatusConflict)
}

func TestTagHandler_RemoveTagFromTask_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	tagID := 15
	called := false

	tagSvc := &mockTagService{
		removeFromTaskFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, tID int, tgID int) error {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, tagID, tgID)
			called = true
			return nil
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/task/tags/"+strconv.Itoa(tagID)+"?taskID="+strconv.Itoa(taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	assert.True(t, called)
}

func TestTagHandler_RemoveTagFromTask_MissingTaskID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	tagID := 15
	tagSvc := &mockTagService{}
	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/task/tags/"+strconv.Itoa(tagID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "missing_task_id", http.StatusBadRequest)
}

func TestTagHandler_RemoveTagFromTask_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	tagID := 15
	tagSvc := &mockTagService{
		removeFromTaskFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, tID int, tgID int) error {
			return service.ErrTagNotFound
		},
	}

	r := setupTagRouter(t, tagSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/task/tags/"+strconv.Itoa(tagID)+"?taskID="+strconv.Itoa(taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "tag_not_found", http.StatusNotFound)
}

func TestTagHandler_Unauthorized_AllEndpoints(t *testing.T) {
	tagSvc := &mockTagService{}
	r := setupTagRouter(t, tagSvc, "test-secret")

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/projects/" + uuid.New().String() + "/tag"},
		{"POST", "/projects/" + uuid.New().String() + "/tag"},
		{"PATCH", "/projects/" + uuid.New().String() + "/tag?tagID=1"},
		{"DELETE", "/projects/" + uuid.New().String() + "/tag?tagID=1"},
		{"GET", "/projects/" + uuid.New().String() + "/task/tags?taskID=1"},
		{"POST", "/projects/" + uuid.New().String() + "/task/tags?taskID=1&tagID=2"}, // 🔹 Обновлено: оба параметра в query
		{"DELETE", "/projects/" + uuid.New().String() + "/task/tags/2?taskID=1"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
