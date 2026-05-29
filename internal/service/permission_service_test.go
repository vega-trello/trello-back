//go:build !integration
// +build !integration

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vega-trello/trello-back/internal/model"
)

var _ PermissionRepository = (*mockPermissionRepository)(nil)

type mockPermissionRepository struct {
	mock.Mock
}

func (m *mockPermissionRepository) GetAllPermissions(ctx context.Context) ([]*model.Permission, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Permission), args.Error(1)
}

func (m *mockPermissionRepository) GetPermissionByID(ctx context.Context, permissionID int) (*model.Permission, error) {
	args := m.Called(ctx, permissionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Permission), args.Error(1)
}

func (m *mockPermissionRepository) GetPermissionsByRoleID(ctx context.Context, roleID int) ([]*model.Permission, error) {
	args := m.Called(ctx, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Permission), args.Error(1)
}

func (m *mockPermissionRepository) GetPermissionByName(ctx context.Context, name string) (*model.Permission, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Permission), args.Error(1)
}

func TestPermissionService_GetAllPermissions_Success(t *testing.T) {
	mockRepo := new(mockPermissionRepository)
	svc := NewPermissionService(mockRepo)
	ctx := context.Background()

	expected := []*model.Permission{
		{ID: 1, Name: "view_project", Description: "Read-only access"},
		{ID: 6, Name: "manage_tasks", Description: "Task management"},
	}

	mockRepo.On("GetAllPermissions", ctx).Return(expected, nil)

	permissions, err := svc.GetAllPermissions(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, permissions)
	mockRepo.AssertExpectations(t)
}

func TestPermissionService_GetAllPermissions_Empty(t *testing.T) {
	mockRepo := new(mockPermissionRepository)
	svc := NewPermissionService(mockRepo)
	ctx := context.Background()

	mockRepo.On("GetAllPermissions", ctx).Return([]*model.Permission{}, nil)

	permissions, err := svc.GetAllPermissions(ctx)

	assert.NoError(t, err)
	assert.Empty(t, permissions)
	mockRepo.AssertExpectations(t)
}

func TestPermissionService_GetAllPermissions_RepoError(t *testing.T) {
	mockRepo := new(mockPermissionRepository)
	svc := NewPermissionService(mockRepo)
	ctx := context.Background()

	repoErr := errors.New("database connection failed")
	mockRepo.On("GetAllPermissions", ctx).Return(nil, repoErr)

	permissions, err := svc.GetAllPermissions(ctx)

	assert.Error(t, err)
	assert.Nil(t, permissions)
	assert.Contains(t, err.Error(), "service: get all permissions")
	mockRepo.AssertExpectations(t)
}
