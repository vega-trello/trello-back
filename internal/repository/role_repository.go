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

var (
	ErrRoleInUse = errors.New("role is in use and cannot be deleted")
)

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
	name string,
	description *string,
	permissionIDs []int,
) (*model.Role, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := r.checkUserAccessTx(ctx, tx, projectUUID, userUUID); err != nil {
		return nil, err
	}

	var role model.Role
	err = tx.QueryRow(ctx, `
		INSERT INTO role (project_uuid, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, project_uuid, name, description
	`, projectUUID, name, description).Scan(
		&role.ID, &role.ProjectUUID, &role.Name, &role.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: create role: %w", err)
	}

	// Привязка разрешений
	for _, permID := range permissionIDs {
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
	if _, err := r.checkUserAccess(ctx, projectUUID, userUUID); err != nil {
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
	var role model.Role
	err := r.db.QueryRow(ctx, `
		SELECT id, project_uuid, name, description 
		FROM role 
		WHERE id = $1 AND (project_uuid = $2 OR project_uuid IS NULL)
	`, roleID, projectUUID).Scan(&role.ID, &role.ProjectUUID, &role.Name, &role.Description)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find role by id: %w", err)
	}
	if role.ProjectUUID != nil {
		if _, err := r.checkUserAccess(ctx, projectUUID, userUUID); err != nil {
			return nil, err
		}
	}
	return &role, nil
}

func (r *RoleRepository) Update(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
	userUUID uuid.UUID,
	name string,
	description *string,
	permissionIDs []int,
) (*model.Role, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var existingProjectUUID *uuid.UUID
	err = tx.QueryRow(ctx, `SELECT project_uuid FROM role WHERE id = $1`, roleID).Scan(&existingProjectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find role: %w", err)
	}
	if existingProjectUUID == nil {
		return nil, ErrCannotDeleteSystemRole
	}
	if _, err := r.checkUserAccessTx(ctx, tx, projectUUID, userUUID); err != nil {
		return nil, err
	}

	var role model.Role
	err = tx.QueryRow(ctx, `
		UPDATE role
		SET name = $1, description = $2
		WHERE id = $3
		RETURNING id, project_uuid, name, description
	`, name, description, roleID).Scan(
		&role.ID, &role.ProjectUUID, &role.Name, &role.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: update role: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM role_permission WHERE role_id = $1`, roleID)
	if err != nil {
		return nil, fmt.Errorf("repository: clear permissions: %w", err)
	}
	for _, permID := range permissionIDs {
		_, err = tx.Exec(ctx, `INSERT INTO role_permission (role_id, permission_id) VALUES ($1, $2)`, roleID, permID)
		if err != nil {
			return nil, fmt.Errorf("repository: assign permission: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &role, nil
}

func (r *RoleRepository) Delete(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
	userUUID uuid.UUID,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var existingProjectUUID *uuid.UUID
	err = tx.QueryRow(ctx, `SELECT project_uuid FROM role WHERE id = $1`, roleID).Scan(&existingProjectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRoleNotFound
	}
	if err != nil {
		return fmt.Errorf("repository: find role: %w", err)
	}
	if existingProjectUUID == nil {
		return ErrCannotDeleteSystemRole
	}
	if _, err := r.checkUserAccessTx(ctx, tx, projectUUID, userUUID); err != nil {
		return err
	}

	var inUse bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_member WHERE role_id = $1)`, roleID).Scan(&inUse)
	if err != nil {
		return fmt.Errorf("repository: check role usage: %w", err)
	}
	if inUse {
		return ErrRoleInUse
	}

	_, err = tx.Exec(ctx, `DELETE FROM role WHERE id = $1`, roleID)
	if err != nil {
		return fmt.Errorf("repository: delete role: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *RoleRepository) FindPermissions(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
	userUUID uuid.UUID,
) ([]*model.Permission, error) {
	_, err := r.FindByID(ctx, projectUUID, roleID, userUUID)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.name, p.description
		FROM permission p
		JOIN role_permission rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.name ASC
	`, roleID)
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

func (r *RoleRepository) checkUserAccess(ctx context.Context, projectUUID, userUUID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_member WHERE project_uuid = $1 AND user_uuid = $2)`, projectUUID, userUUID).Scan(&exists)
	return exists, err
}

func (r *RoleRepository) checkUserAccessTx(ctx context.Context, tx pgx.Tx, projectUUID, userUUID uuid.UUID) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_member WHERE project_uuid = $1 AND user_uuid = $2)`, projectUUID, userUUID).Scan(&exists)
	return exists, err
}
