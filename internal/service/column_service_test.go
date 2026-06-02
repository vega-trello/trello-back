//go:build !integration
// +build !integration

package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	dto "github.com/vega-trello/trello-back/internal/dto/column"
	"github.com/vega-trello/trello-back/internal/model"
)

type MockColumnRepository struct {
	mock.Mock
}

func (m *MockColumnRepository) Create(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, name string, position *int, color *string) (*model.Column, error) {
	args := m.Called(ctx, projectUUID, userUUID, name, position, color)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Column), args.Error(1)
}

func (m *MockColumnRepository) FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Column, error) {
	args := m.Called(ctx, projectUUID, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Column), args.Error(1)
}

func (m *MockColumnRepository) FindByID(ctx context.Context, columnID int, userUUID uuid.UUID) (*model.Column, error) {
	args := m.Called(ctx, columnID, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Column), args.Error(1)
}

func (m *MockColumnRepository) Update(ctx context.Context, columnID int, userUUID uuid.UUID, name string, position *int, color *string) (*model.Column, error) {
	args := m.Called(ctx, columnID, userUUID, name, position, color)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Column), args.Error(1)
}

func (m *MockColumnRepository) Delete(ctx context.Context, columnID int, userUUID uuid.UUID) error {
	args := m.Called(ctx, columnID, userUUID)
	return args.Error(0)
}

func (m *MockColumnRepository) Move(ctx context.Context, columnID int, userUUID uuid.UUID, direction string) (*model.Column, error) {
	args := m.Called(ctx, columnID, userUUID, direction)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Column), args.Error(1)
}

func intPtr(i int) *int { return &i }

func TestColumnService_CreateColumn_Success(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	req := dto.CreateColumnRequest{Name: "To Do", Position: intPtr(0)}

	expected := &model.Column{
		ID:          1,
		ProjectUUID: projectUUID,
		Name:        "To Do",
		Position:    0,
		Color:       nil,
	}

	mockRepo.On("Create", ctx, projectUUID, userUUID, "To Do", intPtr(0), (*string)(nil)).
		Return(expected, nil)

	column, err := svc.CreateColumn(ctx, projectUUID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, expected, column)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_CreateColumn_InvalidName(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	_, err := svc.CreateColumn(ctx, projectUUID, userUUID, dto.CreateColumnRequest{Name: ""})
	assert.ErrorIs(t, err, ErrInvalidColumnName)

	_, err = svc.CreateColumn(ctx, projectUUID, userUUID, dto.CreateColumnRequest{Name: string(make([]byte, 65))})
	assert.ErrorIs(t, err, ErrInvalidColumnName)

	mockRepo.AssertNotCalled(t, "Create")
}

func TestColumnService_CreateColumn_InvalidPosition(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	negPos := -1

	_, err := svc.CreateColumn(ctx, projectUUID, userUUID, dto.CreateColumnRequest{Name: "Test", Position: &negPos})
	assert.ErrorIs(t, err, ErrInvalidPosition)

	mockRepo.AssertNotCalled(t, "Create")
}

func TestColumnService_CreateColumn_AccessDenied(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	req := dto.CreateColumnRequest{Name: "Test"}

	mockRepo.On("Create", ctx, projectUUID, userUUID, "Test", (*int)(nil), (*string)(nil)).
		Return(nil, ErrAccessDenied)

	_, err := svc.CreateColumn(ctx, projectUUID, userUUID, req)

	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_GetProjectColumns_Success(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	expected := []*model.Column{
		{ID: 1, Name: "To Do", Position: 0, Color: nil},
		{ID: 2, Name: "In Progress", Position: 1, Color: nil},
	}

	mockRepo.On("FindByProjectUUID", ctx, projectUUID, userUUID).
		Return(expected, nil)

	columns, err := svc.GetProjectColumns(ctx, projectUUID, userUUID)

	assert.NoError(t, err)
	assert.Equal(t, expected, columns)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_GetProjectColumns_AccessDenied(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("FindByProjectUUID", ctx, projectUUID, userUUID).
		Return(nil, ErrAccessDenied)

	_, err := svc.GetProjectColumns(ctx, projectUUID, userUUID)

	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_GetColumn_Success(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	columnID := 1

	expected := &model.Column{ID: columnID, Name: "Done", Position: 2, Color: nil}

	mockRepo.On("FindByID", ctx, columnID, userUUID).
		Return(expected, nil)

	column, err := svc.GetColumn(ctx, columnID, userUUID)

	assert.NoError(t, err)
	assert.Equal(t, expected, column)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_GetColumn_NotFound(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	mockRepo.On("FindByID", ctx, 1, userUUID).
		Return(nil, ErrAccessDenied)

	_, err := svc.GetColumn(ctx, 1, userUUID)

	assert.ErrorIs(t, err, ErrColumnNotFound)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_GetColumn_RepoNotFound(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	mockRepo.On("FindByID", ctx, 1, userUUID).
		Return(nil, ErrColumnNotFound)

	_, err := svc.GetColumn(ctx, 1, userUUID)

	assert.ErrorIs(t, err, ErrColumnNotFound)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_UpdateColumn_Success(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	columnID := 1

	newName := "Updated"
	newPos := 3
	req := dto.UpdateColumnRequest{
		Name:     newName,
		Position: &newPos,
	}

	expected := &model.Column{ID: columnID, Name: newName, Position: newPos, Color: nil}

	mockRepo.On("Update", ctx, columnID, userUUID, newName, &newPos, (*string)(nil)).
		Return(expected, nil)

	column, err := svc.UpdateColumn(ctx, columnID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, expected, column)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_UpdateColumn_WithColor_Success(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	columnID := 1

	color := "#FF5733"
	req := dto.UpdateColumnRequest{
		Name:  "Updated",
		Color: colorPtr(color),
	}

	expected := &model.Column{ID: columnID, Name: "Updated", Color: &color}

	mockRepo.On("Update", ctx, columnID, userUUID, "Updated", (*int)(nil), colorPtr(color)).
		Return(expected, nil)

	column, err := svc.UpdateColumn(ctx, columnID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, color, *column.Color)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_UpdateColumn_InvalidName(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	_, err := svc.UpdateColumn(ctx, 1, userUUID, dto.UpdateColumnRequest{Name: ""})
	assert.ErrorIs(t, err, ErrInvalidColumnName)

	mockRepo.AssertNotCalled(t, "Update")
}

func TestColumnService_UpdateColumn_NameTooLong(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	_, err := svc.UpdateColumn(ctx, 1, userUUID, dto.UpdateColumnRequest{Name: string(make([]byte, 65))})
	assert.ErrorIs(t, err, ErrInvalidColumnName)

	mockRepo.AssertNotCalled(t, "Update")
}

func TestColumnService_UpdateColumn_InvalidPosition(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	negPos := -1

	_, err := svc.UpdateColumn(ctx, 1, userUUID, dto.UpdateColumnRequest{Name: "Valid", Position: &negPos})
	assert.ErrorIs(t, err, ErrInvalidPosition)

	mockRepo.AssertNotCalled(t, "Update")
}

func TestColumnService_UpdateColumn_InvalidColor(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	invalidColor := "not-a-color"

	_, err := svc.UpdateColumn(ctx, 1, userUUID, dto.UpdateColumnRequest{
		Name:  "Valid",
		Color: stringPtr(invalidColor),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "color must be a valid HEX string")

	mockRepo.AssertNotCalled(t, "Update")
}

func TestColumnService_UpdateColumn_NotFound(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	req := dto.UpdateColumnRequest{Name: "Updated"}

	mockRepo.On("Update", ctx, 1, userUUID, "Updated", (*int)(nil), (*string)(nil)).
		Return(nil, ErrColumnNotFound)

	_, err := svc.UpdateColumn(ctx, 1, userUUID, req)

	assert.ErrorIs(t, err, ErrColumnNotFound)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_UpdateColumn_AccessDenied(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	req := dto.UpdateColumnRequest{Name: "Updated"}

	mockRepo.On("Update", ctx, 1, userUUID, "Updated", (*int)(nil), (*string)(nil)).
		Return(nil, ErrAccessDenied)

	_, err := svc.UpdateColumn(ctx, 1, userUUID, req)

	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_DeleteColumn_Success(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	columnID := 1

	mockRepo.On("Delete", ctx, columnID, userUUID).
		Return(nil)

	err := svc.DeleteColumn(ctx, columnID, userUUID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_DeleteColumn_HasTasks(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	mockRepo.On("Delete", ctx, 1, userUUID).
		Return(ErrColumnHasTasks)

	err := svc.DeleteColumn(ctx, 1, userUUID)

	assert.ErrorIs(t, err, ErrColumnHasTasks)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_DeleteColumn_NotFound(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	mockRepo.On("Delete", ctx, 1, userUUID).
		Return(ErrColumnNotFound)

	err := svc.DeleteColumn(ctx, 1, userUUID)

	assert.ErrorIs(t, err, ErrColumnNotFound)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_DeleteColumn_AccessDenied(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	mockRepo.On("Delete", ctx, 1, userUUID).
		Return(ErrAccessDenied)

	err := svc.DeleteColumn(ctx, 1, userUUID)

	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_MoveColumn_Success_Left(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	columnID := 2

	expected := &model.Column{ID: columnID, Position: 0, Color: nil}

	mockRepo.On("Move", ctx, columnID, userUUID, "left").
		Return(expected, nil)

	column, err := svc.MoveColumn(ctx, columnID, userUUID, "left")

	assert.NoError(t, err)
	assert.Equal(t, expected, column)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_MoveColumn_Success_Right(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	columnID := 1

	expected := &model.Column{ID: columnID, Position: 2, Color: nil}

	mockRepo.On("Move", ctx, columnID, userUUID, "right").
		Return(expected, nil)

	column, err := svc.MoveColumn(ctx, columnID, userUUID, "right")

	assert.NoError(t, err)
	assert.Equal(t, expected, column)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_MoveColumn_InvalidDirection(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	_, err := svc.MoveColumn(ctx, 1, userUUID, "up")
	assert.ErrorIs(t, err, ErrInvalidDirection)

	mockRepo.AssertNotCalled(t, "Move")
}

func TestColumnService_MoveColumn_NotFound(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	mockRepo.On("Move", ctx, 1, userUUID, "left").
		Return(nil, ErrColumnNotFound)

	_, err := svc.MoveColumn(ctx, 1, userUUID, "left")

	assert.ErrorIs(t, err, ErrColumnNotFound)
	mockRepo.AssertExpectations(t)
}

func TestColumnService_MoveColumn_AccessDenied(t *testing.T) {
	mockRepo := new(MockColumnRepository)
	svc := NewColumnService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	mockRepo.On("Move", ctx, 1, userUUID, "left").
		Return(nil, ErrAccessDenied)

	_, err := svc.MoveColumn(ctx, 1, userUUID, "left")

	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}
