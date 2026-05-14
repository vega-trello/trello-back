//go:build !integration
// +build !integration

package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	dto "github.com/vega-trello/trello-back/internal/dto/status"
	"github.com/vega-trello/trello-back/internal/model"
)

type MockStatusRepository struct {
	mock.Mock
}

func (m *MockStatusRepository) Create(ctx context.Context, projectUUID uuid.UUID, name string, callerUUID uuid.UUID) (*model.ProjectStatus, error) {
	args := m.Called(ctx, projectUUID, name, callerUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ProjectStatus), args.Error(1)
}

func (m *MockStatusRepository) FindByProject(ctx context.Context, projectUUID uuid.UUID, callerUUID uuid.UUID) ([]*model.ProjectStatus, error) {
	args := m.Called(ctx, projectUUID, callerUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ProjectStatus), args.Error(1)
}

func (m *MockStatusRepository) FindByID(ctx context.Context, projectUUID uuid.UUID, statusID int, callerUUID uuid.UUID) (*model.ProjectStatus, error) {
	args := m.Called(ctx, projectUUID, statusID, callerUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ProjectStatus), args.Error(1)
}

func (m *MockStatusRepository) Update(ctx context.Context, projectUUID uuid.UUID, statusID int, newName string, callerUUID uuid.UUID) (*model.ProjectStatus, error) {
	args := m.Called(ctx, projectUUID, statusID, newName, callerUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ProjectStatus), args.Error(1)
}

func (m *MockStatusRepository) Delete(ctx context.Context, projectUUID uuid.UUID, statusID int, callerUUID uuid.UUID) error {
	args := m.Called(ctx, projectUUID, statusID, callerUUID)
	return args.Error(0)
}

func TestStatusService_CreateStatus_Success(t *testing.T) {
	mockRepo := new(MockStatusRepository)
	svc := NewStatusService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	req := dto.CreateStatusRequest{Name: "In Progress"}

	expected := &model.ProjectStatus{
		ID:          1,
		ProjectUUID: projectUUID,
		Name:        "In Progress",
	}

	mockRepo.On("Create", ctx, projectUUID, "In Progress", userUUID).
		Return(expected, nil)

	status, err := svc.CreateStatus(ctx, projectUUID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, expected, status)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_CreateStatus_InvalidName(t *testing.T) {
	mockRepo := new(MockStatusRepository)
	svc := NewStatusService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	// Пустое имя
	_, err := svc.CreateStatus(ctx, projectUUID, userUUID, dto.CreateStatusRequest{Name: ""})
	assert.ErrorIs(t, err, ErrInvalidStatusName)

	// Слишком длинное имя
	_, err = svc.CreateStatus(ctx, projectUUID, userUUID, dto.CreateStatusRequest{Name: string(make([]byte, 33))})
	assert.ErrorIs(t, err, ErrInvalidStatusName)

	mockRepo.AssertNotCalled(t, "Create")
}

func TestStatusService_CreateStatus_Duplicate(t *testing.T) {
	mockRepo := new(MockStatusRepository)
	svc := NewStatusService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	req := dto.CreateStatusRequest{Name: "Duplicate"}

	mockRepo.On("Create", ctx, projectUUID, "Duplicate", userUUID).
		Return(nil, ErrStatusAlreadyExists)

	_, err := svc.CreateStatus(ctx, projectUUID, userUUID, req)

	assert.ErrorIs(t, err, ErrStatusAlreadyExists)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_GetProjectStatuses_Success(t *testing.T) {
	mockRepo := new(MockStatusRepository)
	svc := NewStatusService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	expected := []*model.ProjectStatus{
		{ID: 1, Name: "To Do"},
		{ID: 2, Name: "In Progress"},
	}

	mockRepo.On("FindByProject", ctx, projectUUID, userUUID).
		Return(expected, nil)

	statuses, err := svc.GetProjectStatuses(ctx, projectUUID, userUUID)

	assert.NoError(t, err)
	assert.Equal(t, expected, statuses)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_GetProjectStatuses_AccessDenied(t *testing.T) {
	mockRepo := new(MockStatusRepository)
	svc := NewStatusService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("FindByProject", ctx, projectUUID, userUUID).
		Return(nil, ErrAccessDenied)

	_, err := svc.GetProjectStatuses(ctx, projectUUID, userUUID)

	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_GetStatus_Success(t *testing.T) {
	mockRepo := new(MockStatusRepository)
	svc := NewStatusService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	statusID := 1

	expected := &model.ProjectStatus{ID: statusID, Name: "Done"}

	mockRepo.On("FindByID", ctx, projectUUID, statusID, userUUID).
		Return(expected, nil)

	status, err := svc.GetStatus(ctx, projectUUID, statusID, userUUID)

	assert.NoError(t, err)
	assert.Equal(t, expected, status)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_GetStatus_NotFound(t *testing.T) {
	mockRepo := new(MockStatusRepository)
	svc := NewStatusService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	// Репозиторий может вернуть ErrAccessDenied или ErrStatusNotFound — сервис маппит в ErrStatusNotFound
	mockRepo.On("FindByID", ctx, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, ErrAccessDenied)

	_, err := svc.GetStatus(ctx, projectUUID, 1, userUUID)

	assert.ErrorIs(t, err, ErrStatusNotFound)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_UpdateStatus_Success(t *testing.T) {
	mockRepo := new(MockStatusRepository)
	svc := NewStatusService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	statusID := 1
	req := dto.UpdateStatusRequest{Name: "Updated"}

	expected := &model.ProjectStatus{ID: statusID, Name: "Updated"}

	mockRepo.On("Update", ctx, projectUUID, statusID, "Updated", userUUID).
		Return(expected, nil)

	status, err := svc.UpdateStatus(ctx, projectUUID, statusID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, expected, status)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_UpdateStatus_InvalidName(t *testing.T) {
	mockRepo := new(MockStatusRepository)
	svc := NewStatusService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	_, err := svc.UpdateStatus(ctx, projectUUID, 1, userUUID, dto.UpdateStatusRequest{Name: ""})
	assert.ErrorIs(t, err, ErrInvalidStatusName)

	mockRepo.AssertNotCalled(t, "Update")
}

func TestStatusService_UpdateStatus_Duplicate(t *testing.T) {
	mockRepo := new(MockStatusRepository)
	svc := NewStatusService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	req := dto.UpdateStatusRequest{Name: "Taken"}

	mockRepo.On("Update", ctx, projectUUID, 1, "Taken", userUUID).
		Return(nil, ErrStatusAlreadyExists)

	_, err := svc.UpdateStatus(ctx, projectUUID, 1, userUUID, req)

	assert.ErrorIs(t, err, ErrStatusAlreadyExists)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_DeleteStatus_Success(t *testing.T) {
	mockRepo := new(MockStatusRepository)
	svc := NewStatusService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	statusID := 1

	mockRepo.On("Delete", ctx, projectUUID, statusID, userUUID).
		Return(nil)

	err := svc.DeleteStatus(ctx, projectUUID, statusID, userUUID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_DeleteStatus_HasActiveTasks(t *testing.T) {
	mockRepo := new(MockStatusRepository)
	svc := NewStatusService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("Delete", ctx, projectUUID, 1, userUUID).
		Return(ErrStatusHasActiveTasks)

	err := svc.DeleteStatus(ctx, projectUUID, 1, userUUID)

	assert.ErrorIs(t, err, ErrStatusHasActiveTasks)
	mockRepo.AssertExpectations(t)
}

func TestStatusService_DeleteStatus_NotFound(t *testing.T) {
	mockRepo := new(MockStatusRepository)
	svc := NewStatusService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("Delete", ctx, projectUUID, 1, userUUID).
		Return(ErrStatusNotFound)

	err := svc.DeleteStatus(ctx, projectUUID, 1, userUUID)

	assert.ErrorIs(t, err, ErrStatusNotFound)
	mockRepo.AssertExpectations(t)
}
