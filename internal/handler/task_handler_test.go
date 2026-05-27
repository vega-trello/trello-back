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
	"github.com/stretchr/testify/mock"
	dto "github.com/vega-trello/trello-back/internal/dto/task"
	"github.com/vega-trello/trello-back/internal/model"
)

type MockTaskService struct {
	mock.Mock
}

func (m *MockTaskService) CreateTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	req dto.CreateTaskRequest,
) (*model.TaskDB, error) {
	args := m.Called(ctx, projectUUID, userUUID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TaskDB), args.Error(1)
}

func (m *MockTaskService) GetProjectTasks(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	archived *bool,
) ([]*model.TaskDB, error) {
	args := m.Called(ctx, projectUUID, userUUID, archived)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TaskDB), args.Error(1)
}

func (m *MockTaskService) GetTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
) (*model.TaskDB, error) {
	args := m.Called(ctx, projectUUID, taskID, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TaskDB), args.Error(1)
}

func (m *MockTaskService) UpdateTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
	req dto.UpdateTaskRequest,
) (*model.TaskDB, error) {
	args := m.Called(ctx, projectUUID, taskID, userUUID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TaskDB), args.Error(1)
}

func (m *MockTaskService) DeleteTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
) error {
	args := m.Called(ctx, projectUUID, taskID, userUUID)
	return args.Error(0)
}

func (m *MockTaskService) MoveTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	targetColumnID int,
	userUUID uuid.UUID,
) error {
	args := m.Called(ctx, projectUUID, taskID, targetColumnID, userUUID)
	return args.Error(0)
}

func (m *MockTaskService) ArchiveTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
	archive bool,
) error {
	args := m.Called(ctx, projectUUID, taskID, userUUID, archive)
	return args.Error(0)
}

// testAuthMiddleware имитирует middleware.Auth для тестов
func testAuthMiddleware(userUUID uuid.UUID) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Устанавливаем userUUID в контекст (ключ должен совпадать с middleware.GetUserUUID)
		c.Set("userUUID", userUUID)
		c.Next()
	}
}

func setupTestRouter(t *testing.T, service *MockTaskService, userUUID uuid.UUID) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(testAuthMiddleware(userUUID))

	handler := NewTaskHandler(service)
	r.GET("/projects/:projectUUID/tasks", handler.ListProjectTasks)
	r.POST("/projects/:projectUUID/tasks", handler.CreateTask)
	r.GET("/projects/:projectUUID/task", handler.GetTask)
	r.PATCH("/projects/:projectUUID/task", handler.UpdateTask)
	r.DELETE("/projects/:projectUUID/task", handler.DeleteTask)

	return r
}

func createRequest(method, url string, body interface{}) *http.Request {
	var buf *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		buf = bytes.NewBuffer(data)
	} else {
		buf = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, url, buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestTaskHandler_ListProjectTasks_Success(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	mockTasks := []*model.TaskDB{{ID: 1, Title: "Task 1", ColumnID: 1, CreatorUUID: userUUID}}
	mockSvc.On("GetProjectTasks", mock.Anything, projectUUID, userUUID, mock.Anything).
		Return(mockTasks, nil)

	req := createRequest("GET", "/projects/"+projectUUID.String()+"/tasks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestTaskHandler_ListProjectTasks_InvalidProjectUUID(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	req := createRequest("GET", "/projects/invalid-uuid/tasks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "GetProjectTasks")
}

func TestTaskHandler_CreateTask_Success(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	reqBody := dto.CreateTaskRequest{Title: "New Task", ColumnID: intPtr(1)}
	mockTask := &model.TaskDB{
		ID: 1, Title: "New Task", ColumnID: 1,
		CreatorUUID: userUUID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	mockSvc.On("CreateTask", mock.Anything, projectUUID, userUUID, mock.AnythingOfType("dto.CreateTaskRequest")).
		Return(mockTask, nil)

	req := createRequest("POST", "/projects/"+projectUUID.String()+"/tasks", reqBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestTaskHandler_CreateTask_InvalidBody(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	req := createRequest("POST", "/projects/"+projectUUID.String()+"/tasks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "CreateTask")
}

func TestTaskHandler_CreateTask_InvalidColumnID(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	reqBody := dto.CreateTaskRequest{Title: "Bad Task", ColumnID: intPtr(-5)}
	req := createRequest("POST", "/projects/"+projectUUID.String()+"/tasks", reqBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_column_id")
	mockSvc.AssertNotCalled(t, "CreateTask")
}

func TestTaskHandler_GetTask_Success(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	mockTask := &model.TaskDB{
		ID: 42, Title: "Task 42", ColumnID: 1,
		CreatorUUID: userUUID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	mockSvc.On("GetTask", mock.Anything, projectUUID, 42, userUUID).
		Return(mockTask, nil)

	req := createRequest("GET", "/projects/"+projectUUID.String()+"/task?taskID=42", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestTaskHandler_GetTask_MissingTaskID(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	req := createRequest("GET", "/projects/"+projectUUID.String()+"/task", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "missing_param", resp["error"])
	assert.Contains(t, resp["message"], "taskID query parameter is required")
	mockSvc.AssertNotCalled(t, "GetTask")
}

func TestTaskHandler_GetTask_InvalidTaskID(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	req := createRequest("GET", "/projects/"+projectUUID.String()+"/task?taskID=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "invalid_param", resp["error"])
	assert.Contains(t, resp["message"], "taskID must be a positive integer")
	mockSvc.AssertNotCalled(t, "GetTask")
}

func TestTaskHandler_GetTask_NegativeTaskID(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	req := createRequest("GET", "/projects/"+projectUUID.String()+"/task?taskID=-10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "GetTask")
}

func TestTaskHandler_UpdateTask_Success(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	reqBody := dto.UpdateTaskRequest{Title: stringPtr("Updated")}
	mockTask := &model.TaskDB{
		ID: 42, Title: "Updated", ColumnID: 1,
		CreatorUUID: userUUID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	mockSvc.On("UpdateTask", mock.Anything, projectUUID, 42, userUUID, mock.AnythingOfType("dto.UpdateTaskRequest")).
		Return(mockTask, nil)

	req := createRequest("PATCH", "/projects/"+projectUUID.String()+"/task?taskID=42", reqBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestTaskHandler_UpdateTask_MissingTaskID(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	req := createRequest("PATCH", "/projects/"+projectUUID.String()+"/task", dto.UpdateTaskRequest{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "UpdateTask")
}

func TestTaskHandler_UpdateTask_InvalidTaskID(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	req := createRequest("PATCH", "/projects/"+projectUUID.String()+"/task?taskID=xyz", dto.UpdateTaskRequest{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "UpdateTask")
}

func TestTaskHandler_DeleteTask_Success(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	mockSvc.On("DeleteTask", mock.Anything, projectUUID, 99, userUUID).
		Return(nil)

	req := createRequest("DELETE", "/projects/"+projectUUID.String()+"/task?taskID=99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestTaskHandler_DeleteTask_MissingTaskID(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	req := createRequest("DELETE", "/projects/"+projectUUID.String()+"/task", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "DeleteTask")
}

func TestTaskHandler_DeleteTask_InvalidTaskID(t *testing.T) {
	mockSvc := new(MockTaskService)
	userUUID := uuid.New()
	projectUUID := uuid.New()
	r := setupTestRouter(t, mockSvc, userUUID)

	req := createRequest("DELETE", "/projects/"+projectUUID.String()+"/task?taskID=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "DeleteTask")
}

func intPtr(i int) *int          { return &i }
func stringPtr(s string) *string { return &s }
