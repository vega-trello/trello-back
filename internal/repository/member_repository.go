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

type MemberRepositoryInterface interface {
	Create(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, roleID int) (*model.ProjectMember, error)
	FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID) ([]*model.ProjectMember, error)
	Update(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, roleID int) (*model.ProjectMember, error)
	Delete(ctx context.Context, projectUUID, userUUID uuid.UUID) error
	FindByProjectAndUser(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) (*model.ProjectMember, error)
}

type MemberRepository struct {
	db *pgxpool.Pool
}

func NewMemberRepository(db *pgxpool.Pool) *MemberRepository {
	return &MemberRepository{db: db}
}

// POST /projects/{projectUUID}/members
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
	err = tx.QueryRow(ctx,
		`INSERT INTO project_member (project_uuid, user_uuid, role_id, joined_at)
		 VALUES ($1, $2, $3, NOW())
		 RETURNING project_uuid, user_uuid, role_id, joined_at`,
		projectUUID, userUUID, roleID).Scan(
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
			return nil, fmt.Errorf("repository: invalid project, user, or role: %w", err)
		}
		return nil, fmt.Errorf("repository: create member: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &member, nil
}

// GET /projects/{projectUUID}/members
func (r *MemberRepository) FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID) ([]*model.ProjectMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT project_uuid, user_uuid, role_id, joined_at 
		FROM project_member 
		WHERE project_uuid = $1 
		ORDER BY joined_at ASC`,
		projectUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find members by project uuid: %w", err)
	}
	defer rows.Close()

	var members []*model.ProjectMember
	for rows.Next() {
		var member model.ProjectMember
		err := rows.Scan(
			&member.ProjectUUID,
			&member.UserUUID,
			&member.RoleID,
			&member.JoinedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: scan member: %w", err)
		}
		members = append(members, &member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate members: %w", err)
	}

	return members, nil
}

// PATCH /projects/{projectUUID}/member
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
		return nil, fmt.Errorf("repository: update member: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &member, nil
}

// DELETE /projects/{projectUUID}/member
func (r *MemberRepository) Delete(ctx context.Context, projectUUID, userUUID uuid.UUID) error {
	result, err := r.db.Exec(ctx, `
		DELETE FROM project_member 
		WHERE project_uuid = $1 AND user_uuid = $2`,
		projectUUID, userUUID)
	if err != nil {
		return fmt.Errorf("repository: delete member: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func (r *MemberRepository) FindByProjectAndUser(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) (*model.ProjectMember, error) {
	var member model.ProjectMember
	err := r.db.QueryRow(ctx, `
		SELECT project_uuid, user_uuid, role_id, joined_at 
		FROM project_member 
		WHERE project_uuid = $1 AND user_uuid = $2`,
		projectUUID, userUUID).Scan(
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
