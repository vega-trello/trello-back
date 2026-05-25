//go:build !integration
// +build !integration

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	dto "github.com/vega-trello/trello-back/internal/dto/tag"
	"github.com/vega-trello/trello-back/internal/model"
)

type MockTagRepository struct {
	mock.Mock
}

func (m *MockTagRepository) Create(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, name string, color string) (*model.Tag, error) {
	args := m.Called(ctx, projectUUID, userUUID, name, color)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Tag), args.Error(1)
}

func (m *MockTagRepository) FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Tag, error) {
	args := m.Called(ctx, projectUUID, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Tag), args.Error(1)
}

func (m *MockTagRepository) FindByTask(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) ([]*model.Tag, error) {
	args := m.Called(ctx, projectUUID, taskID, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Tag), args.Error(1)
}

func (m *MockTagRepository) Update(ctx context.Context, tagID int, userUUID uuid.UUID, name string, color string) (*model.Tag, error) {
	args := m.Called(ctx, tagID, userUUID, name, color)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Tag), args.Error(1)
}

func (m *MockTagRepository) Delete(ctx context.Context, tagID int, userUUID uuid.UUID) error {
	args := m.Called(ctx, tagID, userUUID)
	return args.Error(0)
}

func (m *MockTagRepository) AddToTask(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, taskID int, tagID int) error {
	args := m.Called(ctx, projectUUID, userUUID, taskID, tagID)
	return args.Error(0)
}

func (m *MockTagRepository) RemoveFromTask(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, taskID int, tagID int) error {
	args := m.Called(ctx, projectUUID, userUUID, taskID, tagID)
	return args.Error(0)
}

func TestTagService_GetProjectTags_Success(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	expected := []*model.Tag{
		{ID: 1, Name: "Bug", Color: "#FF0000"},
		{ID: 2, Name: "Feature", Color: "#00FF00"},
	}

	mockRepo.On("FindByProjectUUID", ctx, projectUUID, userUUID).Return(expected, nil)

	tags, err := svc.GetProjectTags(ctx, projectUUID, userUUID)

	assert.NoError(t, err)
	assert.Equal(t, expected, tags)
	mockRepo.AssertExpectations(t)
}

func TestTagService_GetProjectTags_AccessDenied(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("FindByProjectUUID", ctx, projectUUID, userUUID).Return(nil, ErrAccessDenied)

	_, err := svc.GetProjectTags(ctx, projectUUID, userUUID)

	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}

func TestTagService_GetTaskTags_Success(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1

	expected := []*model.Tag{
		{ID: 1, Name: "Urgent", Color: "#FF0000"},
	}

	mockRepo.On("FindByTask", ctx, projectUUID, taskID, userUUID).Return(expected, nil)

	tags, err := svc.GetTaskTags(ctx, projectUUID, taskID, userUUID)

	assert.NoError(t, err)
	assert.Equal(t, expected, tags)
	mockRepo.AssertExpectations(t)
}

func TestTagService_CreateTag_Success(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	req := dto.CreateTagRequest{
		Name:  "Bug",
		Color: "#FF0000",
	}

	expected := &model.Tag{
		ID:          1,
		ProjectUUID: projectUUID,
		Name:        "Bug",
		Color:       "#FF0000",
	}

	mockRepo.On("Create", ctx, projectUUID, userUUID, "Bug", "#FF0000").Return(expected, nil)

	tag, err := svc.CreateTag(ctx, projectUUID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, expected, tag)
	mockRepo.AssertExpectations(t)
}

func TestTagService_CreateTag_InvalidName(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	_, err := svc.CreateTag(ctx, projectUUID, userUUID, dto.CreateTagRequest{Name: "", Color: "#FF0000"})
	assert.ErrorIs(t, err, ErrInvalidTagName)

	_, err = svc.CreateTag(ctx, projectUUID, userUUID, dto.CreateTagRequest{Name: string(make([]byte, 33)), Color: "#FF0000"})
	assert.ErrorIs(t, err, ErrInvalidTagName)

	mockRepo.AssertNotCalled(t, "Create")
}

func TestTagService_CreateTag_InvalidColor(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	_, err := svc.CreateTag(ctx, projectUUID, userUUID, dto.CreateTagRequest{Name: "Test", Color: "red"})
	assert.ErrorIs(t, err, ErrInvalidColorFormat)

	_, err = svc.CreateTag(ctx, projectUUID, userUUID, dto.CreateTagRequest{Name: "Test", Color: "FF0000"})
	assert.ErrorIs(t, err, ErrInvalidColorFormat)

	mockRepo.AssertNotCalled(t, "Create")
}

func TestTagService_UpdateTag_Success(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	tagID := 1
	req := dto.UpdateTagRequest{
		Name:  "Updated",
		Color: "#00FF00",
	}

	expected := &model.Tag{ID: tagID, Name: "Updated", Color: "#00FF00"}

	mockRepo.On("Update", ctx, tagID, userUUID, "Updated", "#00FF00").Return(expected, nil)

	tag, err := svc.UpdateTag(ctx, tagID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, expected, tag)
	mockRepo.AssertExpectations(t)
}

func TestTagService_UpdateTag_InvalidInput(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	_, err := svc.UpdateTag(ctx, 1, userUUID, dto.UpdateTagRequest{Name: "Test", Color: "invalid"})
	assert.ErrorIs(t, err, ErrInvalidColorFormat)

	mockRepo.AssertNotCalled(t, "Update")
}

func TestTagService_UpdateTag_NotFound(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	req := dto.UpdateTagRequest{Name: "Test", Color: "#FF0000"}

	mockRepo.On("Update", ctx, 1, userUUID, "Test", "#FF0000").Return(nil, ErrTagNotFound)

	_, err := svc.UpdateTag(ctx, 1, userUUID, req)

	assert.ErrorIs(t, err, ErrTagNotFound)
	mockRepo.AssertExpectations(t)
}

func TestTagService_DeleteTag_Success(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	tagID := 1

	mockRepo.On("Delete", ctx, tagID, userUUID).Return(nil)

	err := svc.DeleteTag(ctx, tagID, userUUID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTagService_DeleteTag_NotFound(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	mockRepo.On("Delete", ctx, 1, userUUID).Return(ErrTagNotFound)

	err := svc.DeleteTag(ctx, 1, userUUID)

	assert.ErrorIs(t, err, ErrTagNotFound)
	mockRepo.AssertExpectations(t)
}

func TestTagService_AddTagToTask_Success(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1
	tagID := 2

	mockRepo.On("AddToTask", ctx, projectUUID, userUUID, taskID, tagID).Return(nil)

	err := svc.AddTagToTask(ctx, projectUUID, userUUID, taskID, tagID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTagService_AddTagToTask_InvalidID(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	err := svc.AddTagToTask(ctx, projectUUID, userUUID, 0, 1)
	assert.Error(t, err)

	err = svc.AddTagToTask(ctx, projectUUID, userUUID, 1, 0)
	assert.Error(t, err)

	mockRepo.AssertNotCalled(t, "AddToTask")
}

func TestTagService_AddTagToTask_TagNotInProject(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("AddToTask", ctx, projectUUID, userUUID, 1, 2).
		Return(errors.New("tag does not belong to this project"))

	err := svc.AddTagToTask(ctx, projectUUID, userUUID, 1, 2)

	assert.ErrorIs(t, err, ErrTagNotInProject)
	mockRepo.AssertExpectations(t)
}

func TestTagService_AddTagToTask_TaskNotInProject(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("AddToTask", ctx, projectUUID, userUUID, 1, 2).
		Return(errors.New("task does not belong to this project"))

	err := svc.AddTagToTask(ctx, projectUUID, userUUID, 1, 2)

	assert.ErrorIs(t, err, ErrTaskNotInProject)
	mockRepo.AssertExpectations(t)
}

func TestTagService_AddTagToTask_AlreadyAttached(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("AddToTask", ctx, projectUUID, userUUID, 1, 2).
		Return(errors.New("duplicate key value violates unique constraint"))

	err := svc.AddTagToTask(ctx, projectUUID, userUUID, 1, 2)

	assert.ErrorIs(t, err, ErrTagAlreadyAttached)
	mockRepo.AssertExpectations(t)
}

func TestTagService_AddTagToTask_TagNotFound(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("AddToTask", ctx, projectUUID, userUUID, 1, 2).
		Return(ErrTagNotFound)

	err := svc.AddTagToTask(ctx, projectUUID, userUUID, 1, 2)

	assert.ErrorIs(t, err, ErrTagNotFound)
	mockRepo.AssertExpectations(t)
}

func TestTagService_RemoveTagFromTask_Success(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	taskID := 1
	tagID := 2

	mockRepo.On("RemoveFromTask", ctx, projectUUID, userUUID, taskID, tagID).Return(nil)

	err := svc.RemoveTagFromTask(ctx, projectUUID, userUUID, taskID, tagID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTagService_RemoveTagFromTask_NotFound(t *testing.T) {
	mockRepo := new(MockTagRepository)
	svc := NewTagService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("RemoveFromTask", ctx, projectUUID, userUUID, 1, 2).Return(ErrTagNotFound)

	err := svc.RemoveTagFromTask(ctx, projectUUID, userUUID, 1, 2)

	assert.ErrorIs(t, err, ErrTagNotFound)
	mockRepo.AssertExpectations(t)
}
