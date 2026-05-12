//go:build !integration
// +build !integration

package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vega-trello/trello-back/internal/model"
)

// MockRoleRepository — мок для интерфейса service.RoleRepository
type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) Create(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error) {
	args := m.Called(ctx, projectUUID, userUUID, name, description, permissionIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Role), args.Error(1)
}

func (m *MockRoleRepository) FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Role, error) {
	args := m.Called(ctx, projectUUID, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Role), args.Error(1)
}

func (m *MockRoleRepository) FindByID(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) (*model.Role, error) {
	args := m.Called(ctx, projectUUID, roleID, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Role), args.Error(1)
}

func (m *MockRoleRepository) Update(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error) {
	args := m.Called(ctx, projectUUID, roleID, userUUID, name, description, permissionIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Role), args.Error(1)
}

func (m *MockRoleRepository) Delete(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) error {
	args := m.Called(ctx, projectUUID, roleID, userUUID)
	return args.Error(0)
}

func (m *MockRoleRepository) FindPermissions(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) ([]*model.Permission, error) {
	args := m.Called(ctx, projectUUID, roleID, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Permission), args.Error(1)
}

// ==================== TESTS ====================

func TestRoleService_CreateRole_Success(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	name := "Admin"
	desc := "Full access"
	descPtr := &desc
	perms := []int{1, 2}

	expectedRole := &model.Role{
		ID:          10,
		ProjectUUID: &projectUUID,
		Name:        name,
		Description: descPtr,
	}

	mockRepo.On("Create", ctx, projectUUID, userUUID, name, descPtr, perms).
		Return(expectedRole, nil)

	role, err := svc.CreateRole(ctx, projectUUID, userUUID, name, descPtr, perms)

	assert.NoError(t, err)
	assert.Equal(t, expectedRole, role)
	mockRepo.AssertExpectations(t)
}

func TestRoleService_CreateRole_InvalidName(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	// Пустое имя
	_, err := svc.CreateRole(ctx, projectUUID, userUUID, "", nil, []int{1})
	assert.ErrorIs(t, err, ErrInvalidRoleName)

	// Слишком длинное имя
	_, err = svc.CreateRole(ctx, projectUUID, userUUID, "a_very_long_name_that_exceeds_32_characters_limit", nil, []int{1})
	assert.ErrorIs(t, err, ErrInvalidRoleName)

	mockRepo.AssertNotCalled(t, "Create")
}

func TestRoleService_CreateRole_InvalidDescription(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	longDesc := "a" + string(make([]byte, 256)) // 257 символов

	_, err := svc.CreateRole(ctx, projectUUID, userUUID, "Valid", &longDesc, []int{1})
	assert.ErrorIs(t, err, ErrInvalidDescription)

	mockRepo.AssertNotCalled(t, "Create")
}

func TestRoleService_CreateRole_NoPermissions(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	_, err := svc.CreateRole(ctx, projectUUID, userUUID, "Admin", nil, []int{})
	assert.ErrorIs(t, err, ErrNoPermissions)

	mockRepo.AssertNotCalled(t, "Create")
}

func TestRoleService_GetRole_Success(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	testRoleID := 10

	expectedRole := &model.Role{
		ID:          testRoleID,
		ProjectUUID: &projectUUID,
		Name:        "TestRole",
	}

	mockRepo.On("FindByID", ctx, projectUUID, testRoleID, userUUID).
		Return(expectedRole, nil)

	role, err := svc.GetRole(ctx, projectUUID, testRoleID, userUUID)

	assert.NoError(t, err)
	assert.Equal(t, expectedRole, role)
	mockRepo.AssertExpectations(t)
}

func TestRoleService_GetRole_NotFound(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	testRoleID := 999

	mockRepo.On("FindByID", ctx, projectUUID, testRoleID, userUUID).
		Return(nil, ErrRoleNotFound)

	_, err := svc.GetRole(ctx, projectUUID, testRoleID, userUUID)

	assert.ErrorIs(t, err, ErrRoleNotFound)
	mockRepo.AssertExpectations(t)
}

func TestRoleService_UpdateRole_Success(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	testRoleID := 10
	newName := "Updated"
	newDesc := "New description"
	newDescPtr := &newDesc
	perms := []int{2, 3}

	expectedRole := &model.Role{
		ID:          testRoleID,
		ProjectUUID: &projectUUID,
		Name:        newName,
		Description: newDescPtr,
	}

	mockRepo.On("Update", ctx, projectUUID, testRoleID, userUUID, newName, newDescPtr, perms).
		Return(expectedRole, nil)

	role, err := svc.UpdateRole(ctx, projectUUID, testRoleID, userUUID, newName, newDescPtr, perms)

	assert.NoError(t, err)
	assert.Equal(t, expectedRole, role)
	mockRepo.AssertExpectations(t)
}

func TestRoleService_UpdateRole_SystemRoleProtected(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	systemRoleID := 1 // owner

	mockRepo.On("Update", ctx, projectUUID, systemRoleID, userUUID, "NewName", mock.Anything, mock.Anything).
		Return(nil, ErrCannotDeleteSystemRole)

	_, err := svc.UpdateRole(ctx, projectUUID, systemRoleID, userUUID, "NewName", nil, []int{1})

	assert.ErrorIs(t, err, ErrSystemRoleProtected)
	mockRepo.AssertExpectations(t)
}

func TestRoleService_DeleteRole_Success(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	testRoleID := 10

	mockRepo.On("Delete", ctx, projectUUID, testRoleID, userUUID).
		Return(nil)

	err := svc.DeleteRole(ctx, projectUUID, testRoleID, userUUID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRoleService_DeleteRole_SystemRoleProtected(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	systemRoleID := 1

	mockRepo.On("Delete", ctx, projectUUID, systemRoleID, userUUID).
		Return(ErrCannotDeleteSystemRole)

	err := svc.DeleteRole(ctx, projectUUID, systemRoleID, userUUID)

	assert.ErrorIs(t, err, ErrSystemRoleProtected)
	mockRepo.AssertExpectations(t)
}

func TestRoleService_DeleteRole_RoleInUse(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	testRoleID := 10

	mockRepo.On("Delete", ctx, projectUUID, testRoleID, userUUID).
		Return(ErrRoleInUse)

	err := svc.DeleteRole(ctx, projectUUID, testRoleID, userUUID)

	assert.ErrorIs(t, err, ErrRoleInUse)
	mockRepo.AssertExpectations(t)
}

func TestRoleService_GetRolePermissions_Success(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	testRoleID := 10

	expectedPerms := []*model.Permission{
		{ID: 1, Name: "create_task", Description: "Create tasks"},
		{ID: 2, Name: "update_task", Description: "Update tasks"},
	}

	mockRepo.On("FindPermissions", ctx, projectUUID, testRoleID, userUUID).
		Return(expectedPerms, nil)

	perms, err := svc.GetRolePermissions(ctx, projectUUID, testRoleID, userUUID)

	assert.NoError(t, err)
	assert.Equal(t, expectedPerms, perms)
	mockRepo.AssertExpectations(t)
}
