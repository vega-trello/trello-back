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

// RoleCreateInput отражает тело запроса POST /projects/{projectUUID}/roles
type RoleCreateInput struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	PermissionIDs []int  `json:"permission_ids"`
}

// RoleUpdates отражает тело запроса PATCH /projects/{projectUUID}/roles/{roleID}
// Поля с указателями: nil = не менять
type RoleUpdates struct {
	Name          *string `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
	PermissionIDs *[]int  `json:"permission_ids,omitempty"`
}

type RoleRepositoryInterface interface {
	FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID) ([]*model.Role, error)
	Create(ctx context.Context, projectUUID uuid.UUID, input RoleCreateInput) (*model.Role, error)
	Update(ctx context.Context, projectUUID uuid.UUID, roleID int, updates RoleUpdates) (*model.Role, error)
	Delete(ctx context.Context, projectUUID uuid.UUID, roleID int) error
	FindPermissions(ctx context.Context, projectUUID uuid.UUID, roleID int) ([]*model.Permission, error)
	AssignPermissions(ctx context.Context, projectUUID uuid.UUID, roleID int, permissionIDs []int) error
	HasPermission(ctx context.Context, roleID int, permissionName string) (bool, error)
}

type RoleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{db: db}
}

// FindByProjectUUID возвращает все роли проекта + глобальные роли
func (r *RoleRepository) FindByProjectUUID(
	ctx context.Context,
	projectUUID uuid.UUID,
) ([]*model.Role, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, project_uuid, name, description
		FROM role
		WHERE project_uuid = $1 OR project_uuid IS NULL
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

// Create создаёт новую роль и назначает ей разрешения
func (r *RoleRepository) Create(
	ctx context.Context,
	projectUUID uuid.UUID,
	input RoleCreateInput,
) (*model.Role, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var role model.Role
	err = tx.QueryRow(ctx, `
		INSERT INTO role (project_uuid, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, project_uuid, name, description
	`, projectUUID, input.Name, input.Description).Scan(
		&role.ID, &role.ProjectUUID, &role.Name, &role.Description,
	)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrRoleAlreadyExists
		}
		return nil, fmt.Errorf("repository: create role: %w", err)
	}

	// Назначаем разрешения, если переданы
	if len(input.PermissionIDs) > 0 {
		if err := r.assignPermissionsTx(ctx, tx, role.ID, input.PermissionIDs); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return &role, nil
}

// Update обновляет роль и синхронизирует разрешения
func (r *RoleRepository) Update(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
	updates RoleUpdates,
) (*model.Role, error) {
	if updates.isEmpty() {
		return r.findByID(ctx, projectUUID, roleID)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Проверяем, что роль принадлежит проекту (или глобальная)
	role, err := r.findByID(ctx, projectUUID, roleID)
	if err != nil {
		return nil, err
	}

	// Обновляем поля роли, если переданы
	if updates.Name != nil || updates.Description != nil {
		query := "UPDATE role SET "
		args := []interface{}{}
		paramIdx := 0

		if updates.Name != nil {
			paramIdx++
			query += fmt.Sprintf("name = $%d", paramIdx)
			args = append(args, *updates.Name)
		}
		if updates.Description != nil {
			if paramIdx > 0 {
				query += ", "
			}
			paramIdx++
			query += fmt.Sprintf("description = $%d", paramIdx)
			args = append(args, *updates.Description)
		}

		paramIdx++
		query += fmt.Sprintf(" WHERE id = $%d AND project_uuid = $%d RETURNING id, project_uuid, name, description", paramIdx, paramIdx+1)
		args = append(args, roleID, projectUUID)

		err = tx.QueryRow(ctx, query, args...).Scan(
			&role.ID, &role.ProjectUUID, &role.Name, &role.Description,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		if err != nil {
			if IsUniqueViolation(err) {
				return nil, ErrRoleAlreadyExists
			}
			return nil, fmt.Errorf("repository: update role: %w", err)
		}
	}

	// Обновляем разрешения, если переданы
	if updates.PermissionIDs != nil {
		if err := r.assignPermissionsTx(ctx, tx, roleID, *updates.PermissionIDs); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return role, nil
}

// isEmpty проверяет, пустой ли объект обновлений
func (u RoleUpdates) isEmpty() bool {
	return u.Name == nil && u.Description == nil && u.PermissionIDs == nil
}

// findByID возвращает роль по ID в рамках проекта (внутренний хелпер)
func (r *RoleRepository) findByID(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
) (*model.Role, error) {
	var role model.Role
	err := r.db.QueryRow(ctx, `
		SELECT id, project_uuid, name, description
		FROM role
		WHERE id = $1 AND (project_uuid = $2 OR project_uuid IS NULL)
	`, roleID, projectUUID).Scan(
		&role.ID, &role.ProjectUUID, &role.Name, &role.Description,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find role by id: %w", err)
	}
	return &role, nil
}

// Delete удаляет роль из проекта
// Системные роли (1-4) удалять нельзя
func (r *RoleRepository) Delete(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
) error {
	// Проверяем, что роль не системная
	if roleID <= 4 { // RoleOwner=1, RoleAdmin=2, RoleMember=3, RoleViewer=4
		return ErrCannotDeleteSystemRole
	}
	// Проверяем принадлежность к проекту (или что роль глобальная)
	_, err := r.findByID(ctx, projectUUID, roleID)
	if err != nil {
		return err
	}

	// ON DELETE CASCADE в role_permission автоматически удалит связи
	_, err = r.db.Exec(ctx, `
		DELETE FROM role
		WHERE id = $1 AND project_uuid = $2
	`, roleID, projectUUID)
	if err != nil {
		return fmt.Errorf("repository: delete role: %w", err)
	}

	return nil
}

// FindPermissions возвращает все разрешения, назначенные роли
func (r *RoleRepository) FindPermissions(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
) ([]*model.Permission, error) {
	_, err := r.findByID(ctx, projectUUID, roleID)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.name, p.description
		FROM role_permission rp
		JOIN permission p ON rp.permission_id = p.id
		WHERE rp.role_id = $1
		ORDER BY p.name ASC
	`, roleID)
	if err != nil {
		return nil, fmt.Errorf("repository: find permissions by role: %w", err)
	}
	defer rows.Close()

	var permissions []*model.Permission
	for rows.Next() {
		var perm model.Permission
		if err := rows.Scan(&perm.ID, &perm.Name, &perm.Description); err != nil {
			return nil, fmt.Errorf("repository: scan permission: %w", err)
		}
		permissions = append(permissions, &perm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate permissions: %w", err)
	}

	return permissions, nil
}

// AssignPermissions назначает разрешения роли (полная замена текущего набора)
func (r *RoleRepository) AssignPermissions(
	ctx context.Context,
	projectUUID uuid.UUID,
	roleID int,
	permissionIDs []int,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Проверяем принадлежность роли к проекту
	_, err = r.findByID(ctx, projectUUID, roleID)
	if err != nil {
		return err
	}

	if err := r.assignPermissionsTx(ctx, tx, roleID, permissionIDs); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("repository: commit transaction: %w", err)
	}

	return nil
}

// assignPermissionsTx внутренняя функция для назначения разрешений в транзакции
func (r *RoleRepository) assignPermissionsTx(
	ctx context.Context,
	tx pgx.Tx,
	roleID int,
	permissionIDs []int,
) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM role_permission WHERE role_id = $1
	`, roleID)
	if err != nil {
		return fmt.Errorf("repository: clear role permissions: %w", err)
	}
	if len(permissionIDs) == 0 {
		return nil
	}

	// Проверяем, что все permission_id существуют
	for _, permID := range permissionIDs {
		var exists bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM permission WHERE id = $1)`, permID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("repository: check permission exists: %w", err)
		}
		if !exists {
			return fmt.Errorf("repository: permission %d not found: %w", permID, ErrPermissionNotFound)
		}
	}

	// Используем unnest для эффективной вставки массива
	_, err = tx.Exec(ctx, `
		INSERT INTO role_permission (role_id, permission_id)
		SELECT $1, unnest($2::int[])
	`, roleID, permissionIDs)
	if err != nil {
		return fmt.Errorf("repository: insert role permissions: %w", err)
	}

	return nil
}

// HasPermission проверяет, есть ли у роли конкретное разрешение по имени
func (r *RoleRepository) HasPermission(
	ctx context.Context,
	roleID int,
	permissionName string,
) (bool, error) {
	var has bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM role_permission rp
			JOIN permission p ON rp.permission_id = p.id
			WHERE rp.role_id = $1 AND p.name = $2
		)
	`, roleID, permissionName).Scan(&has)
	if err != nil {
		return false, fmt.Errorf("repository: check permission: %w", err)
	}
	return has, nil
}
