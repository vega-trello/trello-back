package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vega-trello/trello-back/internal/model"
)

type PermissionRepository struct {
	db *pgxpool.Pool
}

func NewPermissionRepository(db *pgxpool.Pool) *PermissionRepository {
	return &PermissionRepository{db: db}
}

// GetAllPermissions возвращает все системные права, доступные в системе
func (r *PermissionRepository) GetAllPermissions(ctx context.Context) ([]*model.Permission, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, description
		FROM permission
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("repository: get all permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*model.Permission
	for rows.Next() {
		var p model.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description); err != nil {
			return nil, fmt.Errorf("repository: scan permission: %w", err)
		}
		permissions = append(permissions, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate permissions: %w", err)
	}

	return permissions, nil
}

// GetPermissionByID находит право по ID
func (r *PermissionRepository) GetPermissionByID(ctx context.Context, permissionID int) (*model.Permission, error) {
	var p model.Permission
	err := r.db.QueryRow(ctx, `
		SELECT id, name, description
		FROM permission
		WHERE id = $1
	`, permissionID).Scan(&p.ID, &p.Name, &p.Description)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPermissionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get permission by id: %w", err)
	}

	return &p, nil
}

// GetPermissionsByRoleID возвращает список прав, назначенных конкретной роли
func (r *PermissionRepository) GetPermissionsByRoleID(ctx context.Context, roleID int) ([]*model.Permission, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.name, p.description
		FROM permission p
		INNER JOIN role_permission rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.id ASC
	`, roleID)
	if err != nil {
		return nil, fmt.Errorf("repository: get permissions by role: %w", err)
	}
	defer rows.Close()

	var permissions []*model.Permission
	for rows.Next() {
		var p model.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description); err != nil {
			return nil, fmt.Errorf("repository: scan permission: %w", err)
		}
		permissions = append(permissions, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate permissions: %w", err)
	}

	return permissions, nil
}

func (r *PermissionRepository) GetPermissionByName(ctx context.Context, name string) (*model.Permission, error) {
	var p model.Permission
	err := r.db.QueryRow(ctx, `
		SELECT id, name, description
		FROM permission
		WHERE name = $1
	`, name).Scan(&p.ID, &p.Name, &p.Description)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPermissionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get permission by name: %w", err)
	}

	return &p, nil
}
