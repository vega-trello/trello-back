package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/vega-trello/trello-back/internal/model"
	"github.com/vega-trello/trello-back/internal/repository"
)

type roleRepoInterface interface {
	GetUserRole(ctx context.Context, projectUUID, userUUID uuid.UUID) (*model.Role, error)
	GetPermissionsByRoleID(ctx context.Context, roleID int) ([]*model.Permission, error)
}

var ErrPermissionDenied = errors.New("permission denied")

type DBPermissionChecker struct {
	roleRepo roleRepoInterface
}

func NewPermissionChecker(roleRepo roleRepoInterface) *DBPermissionChecker {
	return &DBPermissionChecker{roleRepo: roleRepo}
}

func (c *DBPermissionChecker) Check(
	ctx context.Context,
	projectUUID, userUUID uuid.UUID,
	requiredPerm string,
) error {
	role, err := c.roleRepo.GetUserRole(ctx, projectUUID, userUUID)

	if errors.Is(err, repository.ErrRoleNotFound) || role == nil {
		return ErrPermissionDenied
	}
	if err != nil {
		return err
	}

	var userPerms []string
	if role.ProjectUUID == nil {
		userPerms = getSystemRolePermissions(role.ID)
	} else {
		dbPerms, err := c.roleRepo.GetPermissionsByRoleID(ctx, role.ID)
		if err != nil {
			return err
		}
		for _, p := range dbPerms {
			if IsValidPermission(p.Name) {
				userPerms = append(userPerms, p.Name)
			}
		}
	}
	hasPerm := containsPerm(userPerms, requiredPerm)

	if hasPerm {
		return nil
	}

	return ErrPermissionDenied
}

func getSystemRolePermissions(roleID int) []string {
	switch roleID {
	case model.RoleOwner:
		return []string{
			PermViewProject, PermManageProject, PermManageMembers,
			PermManageRoles, PermManageColumns, PermManageTasks,
			PermManageStatuses, PermManageTags, PermManageAssignees,
		}
	case model.RoleAdmin:
		return []string{
			PermViewProject, PermManageProject, PermManageMembers,
			PermManageColumns, PermManageTasks, PermManageStatuses,
			PermManageTags, PermManageAssignees,
		}
	case model.RoleMember:
		return []string{PermViewProject, PermManageTasks, PermManageAssignees}
	case model.RoleViewer:
		return []string{PermViewProject}
	default:
		return []string{}
	}
}

func containsPerm(perms []string, target string) bool {
	for _, p := range perms {
		if p == target {
			return true
		}
	}
	return false
}
