// internal/service/role_service.go
package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/vega-trello/trello-back/internal/model"
)

var (
	ErrInvalidRoleName        = errors.New("role name must be between 1 and 32 characters")
	ErrInvalidDescription     = errors.New("role description must not exceed 256 characters")
	ErrNoPermissions          = errors.New("at least one permission is required")
	ErrSystemRoleProtected    = errors.New("system roles cannot be modified or deleted")
	ErrRoleInUse              = errors.New("role is in use and cannot be deleted")
	ErrRoleNotFound           = errors.New("role not found")
	ErrCannotDeleteSystemRole = errors.New("system roles cannot be modified or deleted")
)

type RoleService struct {
	repo RoleRepository
}

func NewRoleService(repo RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

// CreateRole создаёт новую роль
func (s *RoleService) CreateRole(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	name string,
	description *string,
	permissionIDs []int,
) (*model.Role, error) {
	if name == "" || len(name) > 32 {
		return nil, ErrInvalidRoleName
	}
	if description != nil && len(*description) > 256 {
		return nil, ErrInvalidDescription
	}
	if len(permissionIDs) == 0 {
		return nil, ErrNoPermissions
	}

	return s.repo.Create(ctx, projectUUID, userUUID, name, description, permissionIDs)
}

// GetProjectRoles возвращает все роли проекта, отсортированные по имени
func (s *RoleService) GetProjectRoles(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) ([]*model.Role, error) {
	return s.repo.FindByProjectUUID(ctx, projectUUID, userUUID)
}

// GetRole возвращает роль по ID с проверкой доступа
func (s *RoleService) GetRole(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
	userUUID uuid.UUID,
) (*model.Role, error) {
	role, err := s.repo.FindByID(ctx, projectUUID, roleID, userUUID)
	if err != nil {
		// Маппинг ошибок репозитория в ошибки домена
		if errors.Is(err, ErrRoleNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return role, nil
}

// UpdateRole обновляет роль с полной перезаписью разрешений
func (s *RoleService) UpdateRole(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
	userUUID uuid.UUID,
	name string,
	description *string,
	permissionIDs []int,
) (*model.Role, error) {
	if name == "" || len(name) > 32 {
		return nil, ErrInvalidRoleName
	}
	if description != nil && len(*description) > 256 {
		return nil, ErrInvalidDescription
	}
	if len(permissionIDs) == 0 {
		return nil, ErrNoPermissions
	}

	role, err := s.repo.Update(ctx, projectUUID, roleID, userUUID, name, description, permissionIDs)
	if err != nil {
		if errors.Is(err, ErrCannotDeleteSystemRole) {
			return nil, ErrSystemRoleProtected
		}
		if errors.Is(err, ErrRoleNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return role, nil
}

// DeleteRole удаляет роль с проверкой, что она не используется
func (s *RoleService) DeleteRole(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
	userUUID uuid.UUID,
) error {
	err := s.repo.Delete(ctx, projectUUID, roleID, userUUID)
	if err != nil {
		if errors.Is(err, ErrCannotDeleteSystemRole) {
			return ErrSystemRoleProtected // 403 Forbidden
		}
		if errors.Is(err, ErrRoleInUse) {
			return ErrRoleInUse // 409 Conflict
		}
		if errors.Is(err, ErrRoleNotFound) {
			return ErrRoleNotFound // 404 Not Found
		}
		return err // 500 Internal
	}
	return nil
}

// GetRolePermissions возвращает список разрешений роли
func (s *RoleService) GetRolePermissions(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
	userUUID uuid.UUID,
) ([]*model.Permission, error) {
	return s.repo.FindPermissions(ctx, projectUUID, roleID, userUUID)
}
