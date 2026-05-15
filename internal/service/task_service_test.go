//go:build !integration
// +build !integration

package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	dto "github.com/vega-trello/trello-back/internal/dto/task"
	"github.com/vega-trello/trello-back/internal/model"
)

type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) Create(ctx context.Context, projectUUID uuid.UUID, columnID int, statusID *int, creatorUUID uuid.UUID, title string, description string, startDate *time.Time, endDate *time.Time) (*model.TaskDB, error) {
	args := m.Called(ctx, projectUUID, columnID, statusID, creatorUUID, title, description, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TaskDB), args.Error(1)
}

func (m *MockTaskRepository) FindByID(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) (*model.TaskDB, error) {
	args := m.Called(ctx, projectUUID, taskID, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TaskDB), args.Error(1)
}

func (m *MockTaskRepository) FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, archived *bool) ([]*model.TaskDB, error) {
	args := m.Called(ctx, projectUUID, userUUID, archived)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TaskDB), args.Error(1)
}

func (m *MockTaskRepository) FindByColumn(ctx context.Context, columnID int, userUUID uuid.UUID) ([]*model.TaskDB, error) {
	args := m.Called(ctx, columnID, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TaskDB), args.Error(1)
}

func (m *MockTaskRepository) Update(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, title *string, description *string, startDate *time.Time, endDate *time.Time, columnID *int, statusID *int, archived *bool) (*model.TaskDB, error) {
	args := m.Called(ctx, projectUUID, taskID, userUUID, title, description, startDate, endDate, columnID, statusID, archived)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TaskDB), args.Error(1)
}

func (m *MockTaskRepository) Delete(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) error {
	args := m.Called(ctx, projectUUID, taskID, userUUID)
	return args.Error(0)
}

func (m *MockTaskRepository) Move(ctx context.Context, projectUUID uuid.UUID, taskID int, targetColumnID int, userUUID uuid.UUID) error {
	args := m.Called(ctx, projectUUID, taskID, targetColumnID, userUUID)
	return args.Error(0)
}

func (m *MockTaskRepository) Archive(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, archive bool) error {
	args := m.Called(ctx, projectUUID, taskID, userUUID, archive)
	return args.Error(0)
}

func stringPtr(s string) *string     { return &s }
func boolPtr(b bool) *bool           { return &b }
func timePtr(t time.Time) *time.Time { return &t }

func TestTaskService_CreateTask_Success(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	req := dto.CreateTaskRequest{
		Title:    "New Task",
		ColumnID: intPtr(1),
	}

	expected := &model.TaskDB{
		ID:          1,
		ColumnID:    1,
		CreatorUUID: userUUID,
		Title:       "New Task",
	}

	mockRepo.On("Create", ctx, projectUUID, 1, (*int)(nil), userUUID, "New Task", "", (*time.Time)(nil), (*time.Time)(nil)).
		Return(expected, nil)

	task, err := svc.CreateTask(ctx, projectUUID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, expected, task)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_CreateTask_InvalidTitle(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	_, err := svc.CreateTask(ctx, projectUUID, userUUID, dto.CreateTaskRequest{Title: "", ColumnID: intPtr(1)})
	assert.ErrorIs(t, err, ErrInvalidTitle)

	_, err = svc.CreateTask(ctx, projectUUID, userUUID, dto.CreateTaskRequest{Title: string(make([]byte, 257)), ColumnID: intPtr(1)})
	assert.ErrorIs(t, err, ErrInvalidTitle)

	mockRepo.AssertNotCalled(t, "Create")
}

func TestTaskService_CreateTask_InvalidDateRange(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	start := "2024-04-01T10:00:00Z"
	end := "2024-03-01T10:00:00Z" // end < start

	_, err := svc.CreateTask(ctx, projectUUID, userUUID, dto.CreateTaskRequest{
		Title:     "Task",
		ColumnID:  intPtr(1),
		StartDate: &start,
		EndDate:   &end,
	})
	assert.ErrorIs(t, err, ErrInvalidDateRange)

	mockRepo.AssertNotCalled(t, "Create")
}

func TestTaskService_CreateTask_InvalidDateFormat(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	invalidDate := "not-a-date"

	_, err := svc.CreateTask(ctx, projectUUID, userUUID, dto.CreateTaskRequest{
		Title:     "Task",
		ColumnID:  intPtr(1),
		StartDate: &invalidDate,
	})
	assert.ErrorIs(t, err, ErrInvalidDateFormat)

	mockRepo.AssertNotCalled(t, "Create")
}

func TestTaskService_CreateTask_AccessDenied(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	req := dto.CreateTaskRequest{Title: "Task", ColumnID: intPtr(1)}

	mockRepo.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, ErrAccessDenied)

	_, err := svc.CreateTask(ctx, projectUUID, userUUID, req)

	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_GetProjectTasks_Success(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	expected := []*model.TaskDB{
		{ID: 1, Title: "Task 1"},
		{ID: 2, Title: "Task 2"},
	}

	mockRepo.On("FindByProjectUUID", ctx, projectUUID, userUUID, (*bool)(nil)).
		Return(expected, nil)

	tasks, err := svc.GetProjectTasks(ctx, projectUUID, userUUID, nil)

	assert.NoError(t, err)
	assert.Equal(t, expected, tasks)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_GetTask_Success(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1

	expected := &model.TaskDB{ID: taskID, Title: "Task"}

	mockRepo.On("FindByID", ctx, projectUUID, taskID, userUUID).
		Return(expected, nil)

	task, err := svc.GetTask(ctx, projectUUID, taskID, userUUID)

	assert.NoError(t, err)
	assert.Equal(t, expected, task)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_GetTask_NotFound(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	// Репозиторий может вернуть ErrAccessDenied или ErrTaskNotFound — сервис маппит в ErrTaskNotFound
	mockRepo.On("FindByID", ctx, projectUUID, 1, userUUID).
		Return(nil, ErrAccessDenied)

	_, err := svc.GetTask(ctx, projectUUID, 1, userUUID)

	assert.ErrorIs(t, err, ErrTaskNotFound)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_UpdateTask_Success(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1

	newTitle := "Updated"
	req := dto.UpdateTaskRequest{Title: &newTitle}

	expected := &model.TaskDB{ID: taskID, Title: "Updated"}

	mockRepo.On("Update", ctx, projectUUID, taskID, userUUID, &newTitle, (*string)(nil), (*time.Time)(nil), (*time.Time)(nil), (*int)(nil), (*int)(nil), (*bool)(nil)).
		Return(expected, nil)

	task, err := svc.UpdateTask(ctx, projectUUID, taskID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, expected, task)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_UpdateTask_InvalidTitle(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	empty := ""

	_, err := svc.UpdateTask(ctx, projectUUID, 1, userUUID, dto.UpdateTaskRequest{Title: &empty})
	assert.ErrorIs(t, err, ErrInvalidTitle)

	mockRepo.AssertNotCalled(t, "Update")
}

func TestTaskService_UpdateTask_InvalidDateRange(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	start := "2024-04-01T10:00:00Z"
	end := "2024-03-01T10:00:00Z"

	_, err := svc.UpdateTask(ctx, projectUUID, 1, userUUID, dto.UpdateTaskRequest{
		StartDate: &start,
		EndDate:   &end,
	})
	assert.ErrorIs(t, err, ErrInvalidDateRange)

	mockRepo.AssertNotCalled(t, "Update")
}

func TestTaskService_UpdateTask_Archive(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1

	archived := true
	expected := &model.TaskDB{ID: taskID, ArchivedAt: timePtr(time.Now())}

	mockRepo.On("Update", ctx, projectUUID, taskID, userUUID, (*string)(nil), (*string)(nil), (*time.Time)(nil), (*time.Time)(nil), (*int)(nil), (*int)(nil), &archived).
		Return(expected, nil)

	task, err := svc.UpdateTask(ctx, projectUUID, taskID, userUUID, dto.UpdateTaskRequest{Archived: &archived})

	assert.NoError(t, err)
	assert.NotNil(t, task.ArchivedAt)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_DeleteTask_Success(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1

	mockRepo.On("Delete", ctx, projectUUID, taskID, userUUID).
		Return(nil)

	err := svc.DeleteTask(ctx, projectUUID, taskID, userUUID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_DeleteTask_NotFound(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("Delete", ctx, projectUUID, 1, userUUID).
		Return(ErrTaskNotFound)

	err := svc.DeleteTask(ctx, projectUUID, 1, userUUID)

	assert.ErrorIs(t, err, ErrTaskNotFound)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_MoveTask_Success(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1
	targetColumnID := 2

	mockRepo.On("Move", ctx, projectUUID, taskID, targetColumnID, userUUID).
		Return(nil)

	err := svc.MoveTask(ctx, projectUUID, taskID, targetColumnID, userUUID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_MoveTask_InvalidColumn(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	err := svc.MoveTask(ctx, projectUUID, 1, 0, userUUID)
	assert.ErrorIs(t, err, ErrInvalidColumn)

	mockRepo.AssertNotCalled(t, "Move")
}

func TestTaskService_ArchiveTask_Success(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1

	mockRepo.On("FindByID", ctx, projectUUID, taskID, userUUID).
		Return(&model.TaskDB{ID: taskID}, nil)
	mockRepo.On("Archive", ctx, projectUUID, taskID, userUUID, true).
		Return(nil)

	err := svc.ArchiveTask(ctx, projectUUID, taskID, userUUID, true)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_ArchiveTask_NotFound(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("FindByID", ctx, projectUUID, 1, userUUID).
		Return(nil, ErrTaskNotFound)

	err := svc.ArchiveTask(ctx, projectUUID, 1, userUUID, true)

	assert.ErrorIs(t, err, ErrTaskNotFound)
	mockRepo.AssertExpectations(t)
}
