package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vega-trello/trello-back/internal/model"
)

// Внутренние параметры (не экспортируются, не зависят от DTO)
type CreateRoleParams struct {
	Name          string
	Description   *string
	PermissionIDs []int
}

type UpdateRoleParams struct {
	Name          *string
	Description   *string
	PermissionIDs *[]int
}

type RoleRepositoryInterface interface {
	Create(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, params CreateRoleParams) (*model.Role, error)
	FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Role, error)
	FindByID(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) (*model.Role, error)
	Update(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID, params UpdateRoleParams) (*model.Role, error)
	Delete(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) error
	FindPermissions(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) ([]*model.Permission, error)
	HasPermission(ctx context.Context, roleID int, permissionName string, userUUID uuid.UUID) (bool, error)
}

type RoleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) Create(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	params CreateRoleParams,
) (*model.Role, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.ensureUserAccessTx(ctx, tx, projectUUID, userUUID); err != nil {
		return nil, err
	}

	var role model.Role
	err = tx.QueryRow(ctx, `
		INSERT INTO role (project_uuid, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, project_uuid, name, description
	`, projectUUID, params.Name, params.Description).Scan(
		&role.ID, &role.ProjectUUID, &role.Name, &role.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: create role: %w", err)
	}

	for _, permID := range params.PermissionIDs {
		_, err = tx.Exec(ctx, `INSERT INTO role_permission (role_id, permission_id) VALUES ($1, $2)`, role.ID, permID)
		if err != nil {
			return nil, fmt.Errorf("repository: assign permission: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &role, nil
}

func (r *RoleRepository) FindByProjectUUID(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) ([]*model.Role, error) {
	if err := r.ensureUserAccess(ctx, projectUUID, userUUID); err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, project_uuid, name, description
		FROM role
		WHERE project_uuid = $1
		ORDER BY name ASC
	`, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find roles by project: %w", err)
	}
	defer rows.Close()

	var roles []*model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(&role.ID, &role.ProjectUUID, &role.Name, &role.Description); err != nil {
			return nil, fmt.Errorf("repository: scan role: %w", err)
		}
		roles = append(roles, &role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate roles: %w", err)
	}
	return roles, nil
}

func (r *RoleRepository) FindByID(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
	userUUID uuid.UUID,
) (*model.Role, error) {
	if err := r.ensureUserAccess(ctx, projectUUID, userUUID); err != nil {
		return nil, err
	}

	var role model.Role
	err := r.db.QueryRow(ctx, `
		SELECT id, project_uuid, name, description
		FROM role
		WHERE id = $1 AND project_uuid = $2
	`, roleID, projectUUID).Scan(&role.ID, &role.ProjectUUID, &role.Name, &role.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find role by id: %w", err)
	}
	return &role, nil
}

func (r *RoleRepository) FindPermissions(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
	userUUID uuid.UUID,
) ([]*model.Permission, error) {
	if err := r.ensureUserAccess(ctx, projectUUID, userUUID); err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.name, p.description
		FROM permission p
		JOIN role_permission rp ON p.id = rp.permission_id
		JOIN role r ON rp.role_id = r.id
		WHERE r.id = $1 AND r.project_uuid = $2
		ORDER BY p.name ASC
	`, roleID, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find permissions: %w", err)
	}
	defer rows.Close()

	var perms []*model.Permission
	for rows.Next() {
		var p model.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description); err != nil {
			return nil, fmt.Errorf("repository: scan permission: %w", err)
		}
		perms = append(perms, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate permissions: %w", err)
	}
	return perms, nil
}

func (r *RoleRepository) Update(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
	userUUID uuid.UUID,
	params UpdateRoleParams,
) (*model.Role, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.ensureUserAccessTx(ctx, tx, projectUUID, userUUID); err != nil {
		return nil, err
	}

	query := "UPDATE role SET "
	args := []interface{}{}
	argIdx := 1

	if params.Name != nil {
		args = append(args, *params.Name)
		query += fmt.Sprintf("name = $%d", argIdx)
		argIdx++
	}
	if params.Description != nil {
		if argIdx > 1 {
			query += ", "
		}
		args = append(args, *params.Description)
		query += fmt.Sprintf("description = $%d", argIdx)
		argIdx++
	}

	if params.PermissionIDs != nil {
		_, err = tx.Exec(ctx, `DELETE FROM role_permission WHERE role_id = $1`, roleID)
		if err != nil {
			return nil, fmt.Errorf("repository: clear permissions: %w", err)
		}
		for _, permID := range *params.PermissionIDs {
			_, err = tx.Exec(ctx, `INSERT INTO role_permission (role_id, permission_id) VALUES ($1, $2)`, roleID, permID)
			if err != nil {
				return nil, fmt.Errorf("repository: assign permission: %w", err)
			}
		}
	}

	if argIdx > 1 {
		args = append(args, roleID, projectUUID)
		query += fmt.Sprintf(" WHERE id = $%d AND project_uuid = $%d RETURNING id, project_uuid, name, description", argIdx, argIdx+1)

		var updated model.Role
		err = tx.QueryRow(ctx, query, args...).Scan(&updated.ID, &updated.ProjectUUID, &updated.Name, &updated.Description)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("repository: update role: %w", err)
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("repository: commit transaction: %w", err)
		}
		return &updated, nil
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return r.FindByID(ctx, projectUUID, roleID, userUUID)
}

func (r *RoleRepository) Delete(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
	userUUID uuid.UUID,
) error {
	if err := r.ensureUserAccess(ctx, projectUUID, userUUID); err != nil {
		return err
	}
	if roleID >= 1 && roleID <= 4 {
		return ErrCannotDeleteSystemRole
	}
	result, err := r.db.Exec(ctx, `DELETE FROM role WHERE id = $1 AND project_uuid = $2`, roleID, projectUUID)
	if err != nil {
		return fmt.Errorf("repository: delete role: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrRoleNotFound
	}
	return nil
}

func (r *RoleRepository) HasPermission(
	ctx context.Context,
	roleID int,
	permissionName string,
	userUUID uuid.UUID,
) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM role_permission rp
			JOIN role r ON rp.role_id = r.id
			JOIN permission p ON rp.permission_id = p.id
			JOIN project_member pm ON r.project_uuid = pm.project_uuid
			WHERE r.id = $1 AND p.name = $2 AND pm.user_uuid = $3
		)
	`, roleID, permissionName, userUUID).Scan(&exists)
	return exists, err
}

func (r *RoleRepository) ensureUserAccess(ctx context.Context, projectUUID, userUUID uuid.UUID) error {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_member WHERE project_uuid = $1 AND user_uuid = $2)`, projectUUID, userUUID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("repository: check access: %w", err)
	}
	if !exists {
		return ErrAccessDenied
	}
	return nil
}

func (r *RoleRepository) ensureUserAccessTx(ctx context.Context, tx pgx.Tx, projectUUID, userUUID uuid.UUID) error {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_member WHERE project_uuid = $1 AND user_uuid = $2)`, projectUUID, userUUID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("repository: check access (tx): %w", err)
	}
	if !exists {
		return ErrAccessDenied
	}
	return nil
}
