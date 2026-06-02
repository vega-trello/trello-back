//go:build !integration
// +build !integration

package service

import (
	"context"
	"errors"
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

func (m *MockTaskRepository) Create(
	ctx context.Context,
	projectUUID uuid.UUID,
	columnID int,
	statusID *int,
	creatorUUID uuid.UUID,
	title string,
	description string,
	color *string,
	done bool,
	startDate *time.Time,
	endDate *time.Time,
) (*model.TaskDB, error) {
	args := m.Called(ctx, projectUUID, columnID, statusID, creatorUUID, title, description, color, done, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TaskDB), args.Error(1)
}

func (m *MockTaskRepository) FindByID(
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

func (m *MockTaskRepository) FindByProjectUUID(
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

func (m *MockTaskRepository) FindByColumn(
	ctx context.Context,
	columnID int,
	userUUID uuid.UUID,
) ([]*model.TaskDB, error) {
	args := m.Called(ctx, columnID, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TaskDB), args.Error(1)
}

func (m *MockTaskRepository) Update(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
	title *string,
	description *string,
	startDate *time.Time,
	endDate *time.Time,
	color *string,
	done *bool,
	columnID *int,
	statusID *int,
	archived *bool,
) (*model.TaskDB, error) {
	args := m.Called(ctx, projectUUID, taskID, userUUID, title, description, startDate, endDate, color, done, columnID, statusID, archived)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TaskDB), args.Error(1)
}

func (m *MockTaskRepository) Delete(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
) error {
	args := m.Called(ctx, projectUUID, taskID, userUUID)
	return args.Error(0)
}

func (m *MockTaskRepository) Move(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	columnID int,
	userUUID uuid.UUID,
) error {
	args := m.Called(ctx, projectUUID, taskID, columnID, userUUID)
	return args.Error(0)
}

func (m *MockTaskRepository) Archive(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
	archive bool,
) error {
	args := m.Called(ctx, projectUUID, taskID, userUUID, archive)
	return args.Error(0)
}

func stringPtr(s string) *string     { return &s }
func boolPtr(b bool) *bool           { return &b }
func timePtr(t time.Time) *time.Time { return &t }
func colorPtr(c string) *string      { return &c }

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
		Color:       nil,
		Done:        false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockRepo.On("Create", ctx, projectUUID, 1, (*int)(nil), userUUID, "New Task", "", (*string)(nil), false, (*time.Time)(nil), (*time.Time)(nil)).
		Return(expected, nil)

	task, err := svc.CreateTask(ctx, projectUUID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, expected, task)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_CreateTask_EmptyTitle_Success(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	req := dto.CreateTaskRequest{
		Title:    "",
		ColumnID: intPtr(1),
	}

	expected := &model.TaskDB{
		ID: 1, ColumnID: 1, CreatorUUID: userUUID, Title: "", Color: nil, Done: false,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	mockRepo.On("Create", ctx, projectUUID, 1, (*int)(nil), userUUID, "", "", (*string)(nil), false, (*time.Time)(nil), (*time.Time)(nil)).
		Return(expected, nil)

	task, err := svc.CreateTask(ctx, projectUUID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, "", task.Title)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_CreateTask_InvalidTitle(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	_, err := svc.CreateTask(ctx, projectUUID, userUUID, dto.CreateTaskRequest{
		Title:    string(make([]byte, 257)),
		ColumnID: intPtr(1),
	})
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
	end := "2024-03-01T10:00:00Z"

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

	mockRepo.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
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
		{ID: 1, Title: "Task 1", ColumnID: 1, CreatorUUID: userUUID, Color: nil, Done: false},
		{ID: 2, Title: "Task 2", ColumnID: 2, CreatorUUID: userUUID, Color: nil, Done: false},
	}

	mockRepo.On("FindByProjectUUID", ctx, projectUUID, userUUID, (*bool)(nil)).
		Return(expected, nil)

	tasks, err := svc.GetProjectTasks(ctx, projectUUID, userUUID, nil)

	assert.NoError(t, err)
	assert.Equal(t, expected, tasks)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_GetProjectTasks_WithArchivedFilter(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	archived := true

	expected := []*model.TaskDB{
		{ID: 3, Title: "Archived Task", ArchivedAt: timePtr(time.Now()), Color: nil, Done: false},
	}

	mockRepo.On("FindByProjectUUID", ctx, projectUUID, userUUID, &archived).
		Return(expected, nil)

	tasks, err := svc.GetProjectTasks(ctx, projectUUID, userUUID, &archived)

	assert.NoError(t, err)
	assert.Equal(t, expected, tasks)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_GetProjectTasks_AccessDenied(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("FindByProjectUUID", ctx, projectUUID, userUUID, (*bool)(nil)).
		Return(nil, ErrAccessDenied)

	_, err := svc.GetProjectTasks(ctx, projectUUID, userUUID, nil)

	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_GetTask_Success(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1

	expected := &model.TaskDB{
		ID: taskID, Title: "Task", ColumnID: 1, CreatorUUID: userUUID,
		Color: nil, Done: false,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

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

	mockRepo.On("FindByID", ctx, projectUUID, 1, userUUID).
		Return(nil, ErrAccessDenied)

	_, err := svc.GetTask(ctx, projectUUID, 1, userUUID)

	assert.ErrorIs(t, err, ErrTaskNotFound)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_GetTask_RepoError(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	dbErr := errors.New("database connection failed")
	mockRepo.On("FindByID", ctx, projectUUID, 1, userUUID).
		Return(nil, dbErr)

	_, err := svc.GetTask(ctx, projectUUID, 1, userUUID)

	assert.Error(t, err)
	assert.NotEqual(t, ErrTaskNotFound, err)
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
	req := dto.UpdateTaskRequest{
		Title:    &newTitle,
		ColumnID: intPtr(1),
		Done:     boolPtr(false),
		Archived: boolPtr(false),
	}

	expected := &model.TaskDB{
		ID: taskID, Title: "Updated", ColumnID: 1, CreatorUUID: userUUID,
		Color: nil, Done: false,
		UpdatedAt: time.Now(),
	}

	mockRepo.On("Update", ctx, projectUUID, taskID, userUUID,
		&newTitle, (*string)(nil), (*time.Time)(nil), (*time.Time)(nil),
		(*string)(nil), boolPtr(false),
		intPtr(1), (*int)(nil), boolPtr(false)).
		Return(expected, nil)

	task, err := svc.UpdateTask(ctx, projectUUID, taskID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, expected, task)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_UpdateTask_EmptyTitle_Success(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1

	empty := ""
	req := dto.UpdateTaskRequest{
		Title:    &empty,
		ColumnID: intPtr(1),
		Done:     boolPtr(false),
		Archived: boolPtr(false),
	}

	expected := &model.TaskDB{
		ID: taskID, Title: "", ColumnID: 1, CreatorUUID: userUUID,
		Color: nil, Done: false,
		UpdatedAt: time.Now(),
	}

	mockRepo.On("Update", ctx, projectUUID, taskID, userUUID,
		&empty, (*string)(nil), (*time.Time)(nil), (*time.Time)(nil),
		(*string)(nil), boolPtr(false),
		intPtr(1), (*int)(nil), boolPtr(false)).
		Return(expected, nil)

	task, err := svc.UpdateTask(ctx, projectUUID, taskID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, "", task.Title)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_UpdateTask_WithColorAndDone(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1

	color := "#00FF00"
	done := true

	req := dto.UpdateTaskRequest{
		Color:    colorPtr(color),
		Done:     boolPtr(done),
		ColumnID: intPtr(1),
		Archived: boolPtr(false),
	}

	expected := &model.TaskDB{
		ID: taskID, Title: "Task", ColumnID: 1, CreatorUUID: userUUID,
		Color: &color, Done: done,
		UpdatedAt: time.Now(),
	}

	mockRepo.On("Update", ctx, projectUUID, taskID, userUUID,
		(*string)(nil), (*string)(nil), (*time.Time)(nil), (*time.Time)(nil),
		colorPtr(color), boolPtr(done),
		intPtr(1), (*int)(nil), boolPtr(false)).
		Return(expected, nil)

	task, err := svc.UpdateTask(ctx, projectUUID, taskID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, color, *task.Color)
	assert.True(t, task.Done)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_UpdateTask_InvalidTitle(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	_, err := svc.UpdateTask(ctx, projectUUID, 1, userUUID, dto.UpdateTaskRequest{
		Title:    stringPtr(string(make([]byte, 257))),
		ColumnID: intPtr(1),
		Done:     boolPtr(false),
		Archived: boolPtr(false),
	})
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
		ColumnID:  intPtr(1),
		Done:      boolPtr(false),
		Archived:  boolPtr(false),
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
	expected := &model.TaskDB{
		ID: taskID, Title: "Task", ArchivedAt: timePtr(time.Now()),
		Color: nil, Done: false,
		UpdatedAt: time.Now(),
	}

	req := dto.UpdateTaskRequest{
		Archived: boolPtr(archived),
		ColumnID: intPtr(1),
		Done:     boolPtr(false),
	}

	mockRepo.On("Update", ctx, projectUUID, taskID, userUUID,
		(*string)(nil), (*string)(nil), (*time.Time)(nil), (*time.Time)(nil),
		(*string)(nil), boolPtr(false),
		intPtr(1), (*int)(nil), boolPtr(archived)).
		Return(expected, nil)

	task, err := svc.UpdateTask(ctx, projectUUID, taskID, userUUID, req)

	assert.NoError(t, err)
	assert.NotNil(t, task.ArchivedAt)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_UpdateTask_AccessDenied(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, ErrAccessDenied)

	_, err := svc.UpdateTask(ctx, projectUUID, 1, userUUID, dto.UpdateTaskRequest{
		Title:    stringPtr("Test"),
		ColumnID: intPtr(1),
		Done:     boolPtr(false),
		Archived: boolPtr(false),
	})

	assert.ErrorIs(t, err, ErrAccessDenied)
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

func TestTaskService_DeleteTask_AccessDenied(t *testing.T) {
	mockRepo := new(MockTaskRepository)
	svc := NewTaskService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("Delete", ctx, projectUUID, 1, userUUID).
		Return(ErrAccessDenied)

	err := svc.DeleteTask(ctx, projectUUID, 1, userUUID)

	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}
