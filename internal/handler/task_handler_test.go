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
	dto "github.com/vega-trello/trello-back/internal/dto/task"
	"github.com/vega-trello/trello-back/internal/model"
	"github.com/vega-trello/trello-back/internal/service"
)

type mockTaskService struct {
	createFunc  func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto.CreateTaskRequest) (*model.TaskDB, error)
	listFunc    func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, archived *bool) ([]*model.TaskDB, error)
	getFunc     func(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) (*model.TaskDB, error)
	updateFunc  func(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, req dto.UpdateTaskRequest) (*model.TaskDB, error)
	deleteFunc  func(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) error
	moveFunc    func(ctx context.Context, projectUUID uuid.UUID, taskID int, targetColumnID int, userUUID uuid.UUID) error
	archiveFunc func(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, archive bool) error
}

func (m *mockTaskService) CreateTask(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto.CreateTaskRequest) (*model.TaskDB, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, projectUUID, userUUID, req)
	}
	return nil, nil
}
func (m *mockTaskService) GetProjectTasks(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, archived *bool) ([]*model.TaskDB, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, projectUUID, userUUID, archived)
	}
	return nil, nil
}
func (m *mockTaskService) GetTask(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) (*model.TaskDB, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, projectUUID, taskID, userUUID)
	}
	return nil, nil
}
func (m *mockTaskService) UpdateTask(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, req dto.UpdateTaskRequest) (*model.TaskDB, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, projectUUID, taskID, userUUID, req)
	}
	return nil, nil
}
func (m *mockTaskService) DeleteTask(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, projectUUID, taskID, userUUID)
	}
	return nil
}
func (m *mockTaskService) MoveTask(ctx context.Context, projectUUID uuid.UUID, taskID int, targetColumnID int, userUUID uuid.UUID) error {
	if m.moveFunc != nil {
		return m.moveFunc(ctx, projectUUID, taskID, targetColumnID, userUUID)
	}
	return nil
}
func (m *mockTaskService) ArchiveTask(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, archive bool) error {
	if m.archiveFunc != nil {
		return m.archiveFunc(ctx, projectUUID, taskID, userUUID, archive)
	}
	return nil
}

func setupTaskRouter(t *testing.T, taskSvc *mockTaskService, jwtSecret string) *gin.Engine {
	t.Helper()

	h := NewTaskHandler(taskSvc)

	return SetupTestRouterWithAuth(t, jwtSecret, func(rg *gin.RouterGroup) {
		rg.GET("/projects/:projectUUID/tasks", h.ListProjectTasks)
		rg.POST("/projects/:projectUUID/tasks", h.CreateTask)

		rg.GET("/projects/:projectUUID/task", h.GetTask)
		rg.PATCH("/projects/:projectUUID/task", h.UpdateTask)
		rg.DELETE("/projects/:projectUUID/task", h.DeleteTask)

		rg.POST("/projects/:projectUUID/tasks/:taskID/move", h.MoveTask)
		rg.POST("/projects/:projectUUID/tasks/:taskID/archive", h.ArchiveTask)
	})
}

func TestTaskHandler_ListProjectTasks_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	columnID := 42
	statusID := 1
	now := time.Now()

	taskSvc := &mockTaskService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, archived *bool) ([]*model.TaskDB, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			assert.Nil(t, archived) // 🔹 archived не передан
			return []*model.TaskDB{
				{
					ID:          taskID,
					ColumnID:    columnID,
					StatusID:    &statusID,
					CreatorUUID: userUUID,
					Title:       "Make layout",
					Description: "Task description",
					CreatedAt:   now,
					UpdatedAt:   now,
				},
			}, nil
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var tasks []dto.TaskResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tasks))
	require.Len(t, tasks, 1)
	assert.Equal(t, "Make layout", tasks[0].Title)
	assert.Equal(t, taskID, tasks[0].ID)
}

func TestTaskHandler_ListProjectTasks_WithArchivedFilter(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()

	taskSvc := &mockTaskService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, archived *bool) ([]*model.TaskDB, error) {
			assert.NotNil(t, archived)
			assert.True(t, *archived)
			return []*model.TaskDB{}, nil
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/tasks?archived=true", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTaskHandler_ListProjectTasks_InvalidUUID(t *testing.T) {
	userUUID := uuid.New()
	taskSvc := &mockTaskService{}
	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/not-a-uuid/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_uuid", http.StatusBadRequest)
}

func TestTaskHandler_ListProjectTasks_AccessDenied(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskSvc := &mockTaskService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, archived *bool) ([]*model.TaskDB, error) {
			return nil, service.ErrAccessDenied
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "access_denied", http.StatusForbidden)
}

func TestTaskHandler_CreateTask_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	columnID := 42
	statusID := 1

	taskSvc := &mockTaskService{
		createFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, req dto.CreateTaskRequest) (*model.TaskDB, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, "Make layout", req.Title)
			assert.Equal(t, columnID, *req.ColumnID)
			assert.Equal(t, statusID, *req.StatusID)
			return &model.TaskDB{
				ID:          taskID,
				ColumnID:    columnID,
				StatusID:    &statusID,
				CreatorUUID: userUUID,
				Title:       req.Title,
				Description: req.Description,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{
		"title": "Make layout",
		"description": "Task description",
		"column_id": 42,
		"status_id": 1,
		"start_date": "2024-03-26T10:00:00Z",
		"end_date": "2024-04-01T18:00:00Z"
	}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var task dto.TaskResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &task))
	assert.Equal(t, "Make layout", task.Title)
	assert.Equal(t, columnID, task.ColumnID)
}

func TestTaskHandler_CreateTask_WithoutOptionalFields(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	columnID := 42

	taskSvc := &mockTaskService{
		createFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, req dto.CreateTaskRequest) (*model.TaskDB, error) {
			assert.Nil(t, req.StatusID)
			assert.Nil(t, req.StartDate)
			assert.Nil(t, req.EndDate)
			assert.Equal(t, columnID, *req.ColumnID)
			return &model.TaskDB{
				ID:          taskID,
				ColumnID:    columnID,
				CreatorUUID: userUUID,
				Title:       req.Title,
				CreatedAt:   time.Now(),
			}, nil
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"title": "Simple task", "column_id": 42}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTaskHandler_CreateTask_MissingColumnID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskSvc := &mockTaskService{}
	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"title": "No column"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_column_id", http.StatusBadRequest)
}

func TestTaskHandler_CreateTask_InvalidTitle(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskSvc := &mockTaskService{}
	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"title": "", "column_id": 1}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_request", http.StatusBadRequest)
}

func TestTaskHandler_CreateTask_InvalidColumn(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskSvc := &mockTaskService{
		createFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, req dto.CreateTaskRequest) (*model.TaskDB, error) {
			return nil, service.ErrInvalidColumn
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"title": "Test", "column_id": 999}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/tasks", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_column", http.StatusBadRequest)
}

func TestTaskHandler_GetTask_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	columnID := 42

	taskSvc := &mockTaskService{
		getFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID) (*model.TaskDB, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, userUUID, uUUID)
			return &model.TaskDB{
				ID:          taskID,
				ColumnID:    columnID,
				CreatorUUID: userUUID,
				Title:       "Get this task",
				CreatedAt:   time.Now(),
			}, nil
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/task?taskID="+strconv.Itoa(taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var task dto.TaskResponse
	json.Unmarshal(w.Body.Bytes(), &task)
	assert.Equal(t, taskID, task.ID)
	assert.Equal(t, "Get this task", task.Title)
}

func TestTaskHandler_GetTask_MissingTaskID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskSvc := &mockTaskService{}
	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/task", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "missing_task_id", http.StatusBadRequest)
}

func TestTaskHandler_GetTask_InvalidTaskID(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskSvc := &mockTaskService{}
	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/task?taskID=abc", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_task_id", http.StatusBadRequest)
}

func TestTaskHandler_GetTask_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 999
	taskSvc := &mockTaskService{
		getFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID) (*model.TaskDB, error) {
			return nil, service.ErrTaskNotFound
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/task?taskID="+strconv.Itoa(taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "task_not_found", http.StatusNotFound)
}

func TestTaskHandler_UpdateTask_Success_FullUpdate(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	newTitle := "Updated title"
	newDesc := "Updated description"
	newColumnID := 43

	taskSvc := &mockTaskService{
		updateFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID, req dto.UpdateTaskRequest) (*model.TaskDB, error) {
			assert.Equal(t, taskID, tID)
			assert.Equal(t, newTitle, *req.Title)
			assert.Equal(t, newDesc, *req.Description)
			assert.Equal(t, newColumnID, *req.ColumnID)
			return &model.TaskDB{
				ID:          taskID,
				ColumnID:    newColumnID,
				Title:       newTitle,
				Description: newDesc,
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{
		"title": "Updated title",
		"description": "Updated description",
		"column_id": 43
	}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/task?taskID="+strconv.Itoa(taskID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var task dto.TaskResponse
	json.Unmarshal(w.Body.Bytes(), &task)
	assert.Equal(t, newTitle, task.Title)
	assert.Equal(t, newColumnID, task.ColumnID)
}

func TestTaskHandler_UpdateTask_Success_PartialUpdate_TitleOnly(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	newTitle := "Only title changed"

	taskSvc := &mockTaskService{
		updateFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID, req dto.UpdateTaskRequest) (*model.TaskDB, error) {
			assert.NotNil(t, req.Title)
			assert.Equal(t, newTitle, *req.Title)
			assert.Nil(t, req.Description)
			assert.Nil(t, req.ColumnID)
			return &model.TaskDB{
				ID:        taskID,
				Title:     newTitle,
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"title": "Only title changed"}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/task?taskID="+strconv.Itoa(taskID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var task dto.TaskResponse
	json.Unmarshal(w.Body.Bytes(), &task)
	assert.Equal(t, newTitle, task.Title)
}
func TestTaskHandler_UpdateTask_InvalidDateRange(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105

	taskSvc := &mockTaskService{
		updateFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID, req dto.UpdateTaskRequest) (*model.TaskDB, error) {
			return nil, service.ErrInvalidDateRange
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{
		"start_date": "2024-04-01T18:00:00Z",
		"end_date": "2024-03-26T10:00:00Z"
	}`)
	req := httptest.NewRequest("PATCH", "/projects/"+projectUUID.String()+"/task?taskID="+strconv.Itoa(taskID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_date_range", http.StatusBadRequest)
}

func TestTaskHandler_DeleteTask_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	called := false

	taskSvc := &mockTaskService{
		deleteFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID) error {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, userUUID, uUUID)
			called = true
			return nil
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/task?taskID="+strconv.Itoa(taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	assert.True(t, called)
}

func TestTaskHandler_DeleteTask_NotFound(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 999
	taskSvc := &mockTaskService{
		deleteFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID) error {
			return service.ErrTaskNotFound
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/projects/"+projectUUID.String()+"/task?taskID="+strconv.Itoa(taskID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "task_not_found", http.StatusNotFound)
}

func TestTaskHandler_MoveTask_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105
	targetColumnID := 43

	taskSvc := &mockTaskService{
		moveFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, colID int, uUUID uuid.UUID) error {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, targetColumnID, colID)
			return nil
		},
		getFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID) (*model.TaskDB, error) {
			return &model.TaskDB{
				ID:        taskID,
				ColumnID:  targetColumnID,
				Title:     "Moved task",
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"target_column_id": 43}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/tasks/"+strconv.Itoa(taskID)+"/move", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var task dto.TaskResponse
	json.Unmarshal(w.Body.Bytes(), &task)
	assert.Equal(t, targetColumnID, task.ColumnID)
}

func TestTaskHandler_ArchiveTask_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	taskID := 105

	taskSvc := &mockTaskService{
		archiveFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID, archive bool) error {
			assert.True(t, archive)
			return nil
		},
		getFunc: func(ctx context.Context, pUUID uuid.UUID, tID int, uUUID uuid.UUID) (*model.TaskDB, error) {
			now := time.Now()
			return &model.TaskDB{
				ID:         taskID,
				Title:      "Archived task",
				ArchivedAt: &now,
				UpdatedAt:  now,
			}, nil
		},
	}

	r := setupTaskRouter(t, taskSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"archive": true}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/tasks/"+strconv.Itoa(taskID)+"/archive", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var task dto.TaskResponse
	json.Unmarshal(w.Body.Bytes(), &task)
	assert.NotNil(t, task.ArchivedAt)
}

func TestTaskHandler_Unauthorized_AllEndpoints(t *testing.T) {
	taskSvc := &mockTaskService{}
	r := setupTaskRouter(t, taskSvc, "test-secret")

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/projects/" + uuid.New().String() + "/tasks", ""},
		{"POST", "/projects/" + uuid.New().String() + "/tasks", `{"title":"test","column_id":1}`},
		{"GET", "/projects/" + uuid.New().String() + "/task?taskID=1", ""},
		{"PATCH", "/projects/" + uuid.New().String() + "/task?taskID=1", `{"title":"new"}`},
		{"DELETE", "/projects/" + uuid.New().String() + "/task?taskID=1", ""},
		{"POST", "/projects/" + uuid.New().String() + "/tasks/1/move", `{"target_column_id":2}`},
		{"POST", "/projects/" + uuid.New().String() + "/tasks/1/archive", `{"archive":true}`},
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
