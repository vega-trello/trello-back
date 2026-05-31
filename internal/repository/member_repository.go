package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	dto "github.com/vega-trello/trello-back/internal/dto/member"
	"github.com/vega-trello/trello-back/internal/model"
)

// MemberRepository реализует MemberRepositoryInterface с использованием pgxpool
type MemberRepository struct {
	db *pgxpool.Pool
}

// NewMemberRepository создает новый экземпляр репозитория
func NewMemberRepository(db *pgxpool.Pool) *MemberRepository {
	return &MemberRepository{db: db}
}

// Create добавляет нового участника в проект (с транзакцией для целостности)
func (r *MemberRepository) Create(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	roleID int,
) (*model.ProjectMember, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var member model.ProjectMember
	err = tx.QueryRow(ctx, `
		INSERT INTO project_member (project_uuid, user_uuid, role_id, joined_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING project_uuid, user_uuid, role_id, joined_at
	`, projectUUID, userUUID, roleID).Scan(
		&member.ProjectUUID,
		&member.UserUUID,
		&member.RoleID,
		&member.JoinedAt,
	)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrMemberAlreadyExists
		}
		if IsForeignKeyViolation(err) {
			return nil, fmt.Errorf("repository: invalid project, user, or role reference: %w", err)
		}
		return nil, fmt.Errorf("repository: create member: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return &member, nil
}

func (r *MemberRepository) FindByProjectUUID(
	ctx context.Context,
	projectUUID uuid.UUID,
) ([]*model.ProjectMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT project_uuid, user_uuid, role_id, joined_at
		FROM project_member
		WHERE project_uuid = $1
		ORDER BY joined_at ASC
	`, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find members by project: %w", err)
	}
	defer rows.Close()

	members := []*model.ProjectMember{}

	for rows.Next() {
		var member model.ProjectMember
		if err := rows.Scan(
			&member.ProjectUUID,
			&member.UserUUID,
			&member.RoleID,
			&member.JoinedAt,
		); err != nil {
			return nil, fmt.Errorf("repository: scan member: %w", err)
		}
		members = append(members, &member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate members: %w", err)
	}

	return members, nil
}

// Используется для формирования ответов API (GET /projects/{uuid}/members)
func (r *MemberRepository) FindByProjectUUIDWithDetails(
	ctx context.Context,
	projectUUID uuid.UUID,
) ([]*dto.MemberResponse, error) {
	rows, err := r.db.Query(ctx, `
		SELECT 
			u.username,
			u.uuid AS uuid,           
			pm.project_uuid,
			pm.role_id,
			r.name AS role_name,
			pm.joined_at
		FROM project_member pm
		JOIN base_user u ON pm.user_uuid = u.uuid
		JOIN role r ON pm.role_id = r.id
		WHERE pm.project_uuid = $1
		ORDER BY pm.joined_at ASC
	`, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find members with details: %w", err)
	}
	defer rows.Close()

	members := []*dto.MemberResponse{}

	for rows.Next() {
		var m dto.MemberResponse
		if err := rows.Scan(
			&m.Username,    // u.username
			&m.UUID,        // u.uuid AS uuid (было &m.UserUUID)
			&m.ProjectUUID, // pm.project_uuid
			&m.RoleID,      // pm.role_id
			&m.RoleName,    // r.name AS role_name
			&m.JoinedAt,    // pm.joined_at
		); err != nil {
			return nil, fmt.Errorf("repository: scan member response: %w", err)
		}
		members = append(members, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate member responses: %w", err)
	}

	return members, nil
}

// FindByProjectAndUser находит конкретную запись участника
// Используется для проверки прав: "Является ли этот пользователь участником проекта?"
func (r *MemberRepository) FindByProjectAndUser(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) (*model.ProjectMember, error) {
	var member model.ProjectMember
	err := r.db.QueryRow(ctx, `
		SELECT project_uuid, user_uuid, role_id, joined_at
		FROM project_member
		WHERE project_uuid = $1 AND user_uuid = $2
	`, projectUUID, userUUID).Scan(
		&member.ProjectUUID,
		&member.UserUUID,
		&member.RoleID,
		&member.JoinedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMemberNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find member by project and user: %w", err)
	}
	return &member, nil
}

func (r *MemberRepository) HasRole(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	roleID int,
) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM project_member
			WHERE project_uuid = $1 AND user_uuid = $2 AND role_id = $3
		)
	`, projectUUID, userUUID, roleID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("repository: check role membership: %w", err)
	}
	return exists, nil
}

// Update изменяет роль участника проекта (с транзакцией для целостности)
func (r *MemberRepository) Update(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	roleID int,
) (*model.ProjectMember, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var member model.ProjectMember
	err = tx.QueryRow(ctx, `
		UPDATE project_member
		SET role_id = $1
		WHERE project_uuid = $2 AND user_uuid = $3
		RETURNING project_uuid, user_uuid, role_id, joined_at
	`, roleID, projectUUID, userUUID).Scan(
		&member.ProjectUUID,
		&member.UserUUID,
		&member.RoleID,
		&member.JoinedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMemberNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: update member role: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return &member, nil
}

// Delete удаляет участника из проекта
// Примечание: удаление владельца (role_id=1) должно проверяться в сервисе
func (r *MemberRepository) Delete(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) error {
	result, err := r.db.Exec(ctx, `
		DELETE FROM project_member
		WHERE project_uuid = $1 AND user_uuid = $2
	`, projectUUID, userUUID)
	if err != nil {
		return fmt.Errorf("repository: delete member: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}
