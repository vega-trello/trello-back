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
	"github.com/vega-trello/trello-back/internal/repository"
)

type mockRoleRepo struct {
	mock.Mock
}

func (m *mockRoleRepo) GetUserRole(ctx context.Context, projectUUID, userUUID uuid.UUID) (*model.Role, error) {
	args := m.Called(ctx, projectUUID, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Role), args.Error(1)
}

func (m *mockRoleRepo) GetPermissionsByRoleID(ctx context.Context, roleID int) ([]*model.Permission, error) {
	args := m.Called(ctx, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Permission), args.Error(1)
}

func TestPermissionChecker_Check_NoRole_Denied(t *testing.T) {
	mockRepo := new(mockRoleRepo)
	checker := NewPermissionChecker(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("GetUserRole", ctx, projectUUID, userUUID).
		Return(nil, repository.ErrRoleNotFound)

	err := checker.Check(ctx, projectUUID, userUUID, "view_project")

	assert.ErrorIs(t, err, ErrPermissionDenied)
	mockRepo.AssertExpectations(t)
}

func TestPermissionChecker_Check_NilRole_Denied(t *testing.T) {
	mockRepo := new(mockRoleRepo)
	checker := NewPermissionChecker(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("GetUserRole", ctx, projectUUID, userUUID).
		Return(nil, nil)

	err := checker.Check(ctx, projectUUID, userUUID, "manage_tasks")

	assert.ErrorIs(t, err, ErrPermissionDenied)
	mockRepo.AssertExpectations(t)
}

func TestPermissionChecker_Check_SystemRole_Owner(t *testing.T) {
	mockRepo := new(mockRoleRepo)
	checker := NewPermissionChecker(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	ownerRole := &model.Role{
		ID:          model.RoleOwner,
		ProjectUUID: nil,
		Name:        "Owner",
	}

	mockRepo.On("GetUserRole", ctx, projectUUID, userUUID).
		Return(ownerRole, nil)

	err := checker.Check(ctx, projectUUID, userUUID, "manage_roles")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestPermissionChecker_Check_SystemRole_Viewer_Limited(t *testing.T) {
	mockRepo := new(mockRoleRepo)
	checker := NewPermissionChecker(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	viewerRole := &model.Role{
		ID:          model.RoleViewer,
		ProjectUUID: nil,
		Name:        "Viewer",
	}

	mockRepo.On("GetUserRole", ctx, projectUUID, userUUID).
		Return(viewerRole, nil)

	err := checker.Check(ctx, projectUUID, userUUID, "view_project")
	assert.NoError(t, err)

	err = checker.Check(ctx, projectUUID, userUUID, "manage_tasks")
	assert.ErrorIs(t, err, ErrPermissionDenied)

	mockRepo.AssertExpectations(t)
}

func TestPermissionChecker_Check_SystemRole_UnknownID(t *testing.T) {
	mockRepo := new(mockRoleRepo)
	checker := NewPermissionChecker(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	unknownRole := &model.Role{
		ID:          999,
		ProjectUUID: nil,
		Name:        "Ghost",
	}

	mockRepo.On("GetUserRole", ctx, projectUUID, userUUID).
		Return(unknownRole, nil)

	err := checker.Check(ctx, projectUUID, userUUID, "view_project")

	assert.ErrorIs(t, err, ErrPermissionDenied)
	mockRepo.AssertExpectations(t)
}

func TestPermissionChecker_Check_CustomRole_HasPerm(t *testing.T) {
	mockRepo := new(mockRoleRepo)
	checker := NewPermissionChecker(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	customRole := &model.Role{
		ID:          100,
		ProjectUUID: &projectUUID,
		Name:        "Custom Editor",
	}

	mockRepo.On("GetUserRole", ctx, projectUUID, userUUID).
		Return(customRole, nil)

	perms := []*model.Permission{
		{ID: 1, Name: "view_project"},
		{ID: 2, Name: "manage_tasks"},
	}
	mockRepo.On("GetPermissionsByRoleID", ctx, 100).
		Return(perms, nil)

	err := checker.Check(ctx, projectUUID, userUUID, "manage_tasks")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestPermissionChecker_Check_CustomRole_NoPerm(t *testing.T) {
	mockRepo := new(mockRoleRepo)
	checker := NewPermissionChecker(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	customRole := &model.Role{
		ID:          101,
		ProjectUUID: &projectUUID,
		Name:        "Limited Role",
	}

	mockRepo.On("GetUserRole", ctx, projectUUID, userUUID).
		Return(customRole, nil)

	perms := []*model.Permission{
		{ID: 1, Name: "view_project"},
	}
	mockRepo.On("GetPermissionsByRoleID", ctx, 101).
		Return(perms, nil)

	err := checker.Check(ctx, projectUUID, userUUID, "manage_roles")

	assert.ErrorIs(t, err, ErrPermissionDenied)
	mockRepo.AssertExpectations(t)
}

func TestPermissionChecker_Check_CustomRole_RepoError(t *testing.T) {
	mockRepo := new(mockRoleRepo)
	checker := NewPermissionChecker(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	customRole := &model.Role{
		ID:          102,
		ProjectUUID: &projectUUID,
		Name:        "Broken Role",
	}

	mockRepo.On("GetUserRole", ctx, projectUUID, userUUID).
		Return(customRole, nil)

	permErr := errors.New("database connection failed")
	mockRepo.On("GetPermissionsByRoleID", ctx, 102).
		Return(nil, permErr)

	err := checker.Check(ctx, projectUUID, userUUID, "view_project")

	assert.Error(t, err)
	assert.NotEqual(t, ErrPermissionDenied, err)
	assert.Contains(t, err.Error(), "database connection failed")
	mockRepo.AssertExpectations(t)
}

func TestPermissionChecker_Check_CustomRole_InvalidPermission(t *testing.T) {
	mockRepo := new(mockRoleRepo)
	checker := NewPermissionChecker(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	customRole := &model.Role{
		ID:          103,
		ProjectUUID: &projectUUID,
		Name:        "Legacy Role",
	}

	mockRepo.On("GetUserRole", ctx, projectUUID, userUUID).
		Return(customRole, nil)

	perms := []*model.Permission{
		{ID: 1, Name: "view_project"},
		{ID: 999, Name: "delete_all"},
	}
	mockRepo.On("GetPermissionsByRoleID", ctx, 103).
		Return(perms, nil)

	err := checker.Check(ctx, projectUUID, userUUID, "view_project")
	assert.NoError(t, err)

	err = checker.Check(ctx, projectUUID, userUUID, "delete_all")
	assert.ErrorIs(t, err, ErrPermissionDenied)

	mockRepo.AssertExpectations(t)
}

func TestPermissionChecker_Check_CustomRole_EmptyPermissions(t *testing.T) {
	mockRepo := new(mockRoleRepo)
	checker := NewPermissionChecker(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	customRole := &model.Role{
		ID:          104,
		ProjectUUID: &projectUUID,
		Name:        "Empty Role",
	}

	mockRepo.On("GetUserRole", ctx, projectUUID, userUUID).
		Return(customRole, nil)

	mockRepo.On("GetPermissionsByRoleID", ctx, 104).
		Return([]*model.Permission{}, nil)

	err := checker.Check(ctx, projectUUID, userUUID, "view_project")

	assert.ErrorIs(t, err, ErrPermissionDenied)
	mockRepo.AssertExpectations(t)
}

func TestContainsPerm_Found(t *testing.T) {
	perms := []string{"view_project", "manage_tasks"}
	assert.True(t, containsPerm(perms, "manage_tasks"))
	assert.True(t, containsPerm(perms, "view_project"))
}

func TestContainsPerm_NotFound(t *testing.T) {
	perms := []string{"view_project", "manage_tasks"}
	assert.False(t, containsPerm(perms, "manage_roles"))
	assert.False(t, containsPerm(perms, ""))
}

func TestContainsPerm_EmptySlice(t *testing.T) {
	perms := []string{}
	assert.False(t, containsPerm(perms, "view_project"))
}

func TestGetSystemRolePermissions_AllRoles(t *testing.T) {
	roles := []int{model.RoleOwner, model.RoleAdmin, model.RoleMember, model.RoleViewer}
	for _, roleID := range roles {
		perms := getSystemRolePermissions(roleID)
		assert.NotEmpty(t, perms, "Role %d should have permissions", roleID)
	}

	perms := getSystemRolePermissions(999)
	assert.Empty(t, perms)
}
