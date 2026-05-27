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
	"github.com/vega-trello/trello-back/internal/model"
)

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

func (m *MockRoleRepository) GetPermissionNamesByID(ctx context.Context, permissionIDs []int) (map[int]string, error) {
	args := m.Called(ctx, permissionIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[int]string), args.Error(1)
}

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

	permNames := map[int]string{1: "manage_tasks", 2: "view_project"}
	mockRepo.On("GetPermissionNamesByID", ctx, perms).Return(permNames, nil)

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

	_, err := svc.CreateRole(ctx, projectUUID, userUUID, "", nil, []int{1})
	assert.ErrorIs(t, err, ErrInvalidRoleName)

	_, err = svc.CreateRole(ctx, projectUUID, userUUID, "a_very_long_name_that_exceeds_32_characters_limit", nil, []int{1})
	assert.ErrorIs(t, err, ErrInvalidRoleName)

	mockRepo.AssertNotCalled(t, "GetPermissionNamesByID")
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

	mockRepo.AssertNotCalled(t, "GetPermissionNamesByID")
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

	mockRepo.AssertNotCalled(t, "GetPermissionNamesByID")
	mockRepo.AssertNotCalled(t, "Create")
}

func TestRoleService_CreateRole_InvalidPermission_NotInEnum(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	perms := []int{999}

	permNames := map[int]string{999: "delete_all"}
	mockRepo.On("GetPermissionNamesByID", ctx, perms).Return(permNames, nil)

	_, err := svc.CreateRole(ctx, projectUUID, userUUID, "BadRole", nil, perms)

	assert.ErrorIs(t, err, ErrInvalidPermission)
	assert.Contains(t, err.Error(), "permission ID 999 has name 'delete_all'")
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Create")
}

func TestRoleService_CreateRole_InvalidPermission_RepoError(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	perms := []int{1}

	permErr := errors.New("database connection failed")
	mockRepo.On("GetPermissionNamesByID", ctx, perms).Return(nil, permErr)

	_, err := svc.CreateRole(ctx, projectUUID, userUUID, "Role", nil, perms)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validate permissions")
	assert.Contains(t, err.Error(), "database connection failed")
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Create")
}

func TestRoleService_CreateRole_InvalidPermission_NotFound(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	perms := []int{99999}

	permErr := errors.New("repository: permission ID 99999 not found")
	mockRepo.On("GetPermissionNamesByID", ctx, perms).Return(nil, permErr)

	_, err := svc.CreateRole(ctx, projectUUID, userUUID, "Role", nil, perms)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validate permissions")
	assert.Contains(t, err.Error(), "permission ID 99999 not found")
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Create")
}

func TestRoleService_CreateRole_ValidPermissions_AllStandard(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	perms := []int{1, 2, 3}

	permNames := map[int]string{
		1: "manage_tasks",
		2: "view_project",
		3: "manage_members",
	}
	mockRepo.On("GetPermissionNamesByID", ctx, perms).Return(permNames, nil)

	expectedRole := &model.Role{ID: 10, ProjectUUID: &projectUUID, Name: "ValidRole"}
	mockRepo.On("Create", ctx, projectUUID, userUUID, "ValidRole", mock.Anything, perms).
		Return(expectedRole, nil)

	role, err := svc.CreateRole(ctx, projectUUID, userUUID, "ValidRole", nil, perms)

	assert.NoError(t, err)
	assert.Equal(t, expectedRole, role)
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

	permNames := map[int]string{2: "view_project", 3: "manage_members"}
	mockRepo.On("GetPermissionNamesByID", ctx, perms).Return(permNames, nil)

	mockRepo.On("Update", ctx, projectUUID, testRoleID, userUUID, newName, newDescPtr, perms).
		Return(expectedRole, nil)

	role, err := svc.UpdateRole(ctx, projectUUID, testRoleID, userUUID, newName, newDescPtr, perms)

	assert.NoError(t, err)
	assert.Equal(t, expectedRole, role)
	mockRepo.AssertExpectations(t)
}

func TestRoleService_UpdateRole_InvalidPermission(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	testRoleID := 10
	perms := []int{999}

	permNames := map[int]string{999: "delete_all"}
	mockRepo.On("GetPermissionNamesByID", ctx, perms).Return(permNames, nil)

	_, err := svc.UpdateRole(ctx, projectUUID, testRoleID, userUUID, "NewName", nil, perms)

	assert.ErrorIs(t, err, ErrInvalidPermission)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Update")
}

func TestRoleService_UpdateRole_SystemRoleProtected(t *testing.T) {
	mockRepo := new(MockRoleRepository)
	svc := NewRoleService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	systemRoleID := 1 // owner

	permNames := map[int]string{1: "view_project"}
	mockRepo.On("GetPermissionNamesByID", ctx, []int{1}).Return(permNames, nil)

	mockRepo.On("Update", ctx, projectUUID, systemRoleID, userUUID, "NewName", mock.Anything, mock.Anything).
		Return(nil, ErrCannotDeleteSystemRole)

	_, err := svc.UpdateRole(ctx, projectUUID, systemRoleID, userUUID, "NewName", nil, []int{1})

	assert.ErrorIs(t, err, ErrSystemRoleProtected)
	mockRepo.AssertExpectations(t)
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
		{ID: 1, Name: "manage_tasks", Description: "Create tasks"},
		{ID: 2, Name: "view_project", Description: "View project"},
	}

	mockRepo.On("FindPermissions", ctx, projectUUID, testRoleID, userUUID).
		Return(expectedPerms, nil)

	perms, err := svc.GetRolePermissions(ctx, projectUUID, testRoleID, userUUID)

	assert.NoError(t, err)
	assert.Equal(t, expectedPerms, perms)
	mockRepo.AssertExpectations(t)
}
