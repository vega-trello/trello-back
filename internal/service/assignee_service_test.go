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
	dto "github.com/vega-trello/trello-back/internal/dto/assignee"
	repoPkg "github.com/vega-trello/trello-back/internal/repository"
)

type MockAssigneeRepository struct {
	mock.Mock
}

func (m *MockAssigneeRepository) Add(ctx context.Context, projectUUID uuid.UUID, taskID int, assigneeUUID uuid.UUID, callerUUID uuid.UUID) error {
	args := m.Called(ctx, projectUUID, taskID, assigneeUUID, callerUUID)
	return args.Error(0)
}

func (m *MockAssigneeRepository) Remove(ctx context.Context, projectUUID uuid.UUID, taskID int, assigneeUUID uuid.UUID, callerUUID uuid.UUID) error {
	args := m.Called(ctx, projectUUID, taskID, assigneeUUID, callerUUID)
	return args.Error(0)
}

func (m *MockAssigneeRepository) FindByTask(ctx context.Context, projectUUID uuid.UUID, taskID int, callerUUID uuid.UUID) ([]*repoPkg.AssigneeResponse, error) {
	args := m.Called(ctx, projectUUID, taskID, callerUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repoPkg.AssigneeResponse), args.Error(1)
}

func TestAssigneeService_GetTaskAssignees_Success(t *testing.T) {
	mockRepo := new(MockAssigneeRepository)
	svc := NewAssigneeService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1

	repoAssignees := []*repoPkg.AssigneeResponse{
		{TaskID: taskID, UserUUID: uuid.New(), Username: "alice", AssignedAt: time.Now()},
		{TaskID: taskID, UserUUID: uuid.New(), Username: "bob", AssignedAt: time.Now()},
	}

	mockRepo.On("FindByTask", ctx, projectUUID, taskID, userUUID).Return(repoAssignees, nil)

	assignees, err := svc.GetTaskAssignees(ctx, projectUUID, taskID, userUUID)

	assert.NoError(t, err)
	assert.Len(t, assignees, 2)
	assert.Equal(t, "alice", assignees[0].User.Username)
	assert.Equal(t, "bob", assignees[1].User.Username)
	assert.Equal(t, repoAssignees[0].UserUUID.String(), assignees[0].UserUUID)
	mockRepo.AssertExpectations(t)
}

func TestAssigneeService_GetTaskAssignees_AccessDenied(t *testing.T) {
	mockRepo := new(MockAssigneeRepository)
	svc := NewAssigneeService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("FindByTask", ctx, projectUUID, 1, userUUID).Return(nil, repoPkg.ErrAccessDenied)

	_, err := svc.GetTaskAssignees(ctx, projectUUID, 1, userUUID)
	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}

func TestAssigneeService_GetTaskAssignees_TaskNotFound(t *testing.T) {
	mockRepo := new(MockAssigneeRepository)
	svc := NewAssigneeService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("FindByTask", ctx, projectUUID, 999, userUUID).Return(nil, repoPkg.ErrTaskNotFound)

	_, err := svc.GetTaskAssignees(ctx, projectUUID, 999, userUUID)
	assert.ErrorIs(t, err, ErrTaskNotFound)
	mockRepo.AssertExpectations(t)
}

func TestAssigneeService_AssignUserToTask_Success(t *testing.T) {
	mockRepo := new(MockAssigneeRepository)
	svc := NewAssigneeService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1
	assigneeUUID := uuid.New()
	req := dto.CreateAssigneeRequest{UserUUID: assigneeUUID.String()}

	mockRepo.On("Add", ctx, projectUUID, taskID, assigneeUUID, userUUID).Return(nil)

	err := svc.AssignUserToTask(ctx, projectUUID, taskID, userUUID, req)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAssigneeService_AssignUserToTask_InvalidUUID(t *testing.T) {
	mockRepo := new(MockAssigneeRepository)
	svc := NewAssigneeService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	req := dto.CreateAssigneeRequest{UserUUID: "not-a-uuid"}

	err := svc.AssignUserToTask(ctx, projectUUID, 1, userUUID, req)
	assert.ErrorIs(t, err, ErrInvalidUserUUID)
	mockRepo.AssertNotCalled(t, "Add")
}

func TestAssigneeService_AssignUserToTask_AlreadyAssigned(t *testing.T) {
	mockRepo := new(MockAssigneeRepository)
	svc := NewAssigneeService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	assigneeUUID := uuid.New()
	req := dto.CreateAssigneeRequest{UserUUID: assigneeUUID.String()}

	mockRepo.On("Add", ctx, projectUUID, 1, assigneeUUID, userUUID).Return(repoPkg.ErrAlreadyAssigned)

	err := svc.AssignUserToTask(ctx, projectUUID, 1, userUUID, req)
	assert.ErrorIs(t, err, ErrAlreadyAssigned)
	mockRepo.AssertExpectations(t)
}

func TestAssigneeService_AssignUserToTask_UserNotFound(t *testing.T) {
	mockRepo := new(MockAssigneeRepository)
	svc := NewAssigneeService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	assigneeUUID := uuid.New()
	req := dto.CreateAssigneeRequest{UserUUID: assigneeUUID.String()}

	mockRepo.On("Add", ctx, projectUUID, 1, assigneeUUID, userUUID).Return(repoPkg.ErrUserNotFound)

	err := svc.AssignUserToTask(ctx, projectUUID, 1, userUUID, req)
	assert.ErrorIs(t, err, ErrUserNotFound)
	mockRepo.AssertExpectations(t)
}

func TestAssigneeService_RemoveAssignee_Success(t *testing.T) {
	mockRepo := new(MockAssigneeRepository)
	svc := NewAssigneeService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1
	assigneeUUID := uuid.New()

	mockRepo.On("Remove", ctx, projectUUID, taskID, assigneeUUID, userUUID).Return(nil)

	err := svc.RemoveAssignee(ctx, projectUUID, taskID, userUUID, assigneeUUID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAssigneeService_RemoveAssignee_NotFound(t *testing.T) {
	mockRepo := new(MockAssigneeRepository)
	svc := NewAssigneeService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	assigneeUUID := uuid.New()

	mockRepo.On("Remove", ctx, projectUUID, 1, assigneeUUID, userUUID).Return(repoPkg.ErrAssigneeNotFound)

	err := svc.RemoveAssignee(ctx, projectUUID, 1, userUUID, assigneeUUID)
	assert.ErrorIs(t, err, ErrAssigneeNotFound)
	mockRepo.AssertExpectations(t)
}

func TestAssigneeService_RemoveAssignee_AccessDenied(t *testing.T) {
	mockRepo := new(MockAssigneeRepository)
	svc := NewAssigneeService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	assigneeUUID := uuid.New()

	mockRepo.On("Remove", ctx, projectUUID, 1, assigneeUUID, userUUID).Return(repoPkg.ErrAccessDenied)

	err := svc.RemoveAssignee(ctx, projectUUID, 1, userUUID, assigneeUUID)
	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}
