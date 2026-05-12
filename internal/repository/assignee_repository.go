package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AssigneeResponse структура ответа API (task_assignee + данные пользователя)
type AssigneeResponse struct {
	TaskID     int       `json:"task_id"`
	UserUUID   uuid.UUID `json:"user_uuid"`
	Username   string    `json:"username"`
	AssignedAt time.Time `json:"assigned_at"`
}

type AssigneeRepository struct {
	db *pgxpool.Pool
}

func NewAssigneeRepository(db *pgxpool.Pool) *AssigneeRepository {
	return &AssigneeRepository{db: db}
}

func (r *AssigneeRepository) Add(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	assigneeUUID uuid.UUID,
	callerUUID uuid.UUID,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.ensureUserAccessTx(ctx, tx, projectUUID, callerUUID); err != nil {
		return err
	}

	var taskProjectUUID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT pc.project_uuid 
		FROM tasks t 
		JOIN project_column pc ON t.column_id = pc.id 
		WHERE t.id = $1
	`, taskID).Scan(&taskProjectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrTaskNotFound
	}
	if err != nil {
		return fmt.Errorf("repository: check task project: %w", err)
	}
	if taskProjectUUID != projectUUID {
		return ErrTaskNotFound
	}

	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM base_user WHERE uuid = $1)`, assigneeUUID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("repository: check assignee user: %w", err)
	}
	if !exists {
		return ErrUserNotFound
	}

	result, err := tx.Exec(ctx, `
		INSERT INTO task_assignee (task_id, user_uuid, assigned_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (task_id, user_uuid) DO NOTHING
	`, taskID, assigneeUUID, time.Now())
	if err != nil {
		return fmt.Errorf("repository: insert assignee: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAlreadyAssigned
	}

	return tx.Commit(ctx)
}

func (r *AssigneeRepository) Remove(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	assigneeUUID uuid.UUID,
	callerUUID uuid.UUID,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.ensureUserAccessTx(ctx, tx, projectUUID, callerUUID); err != nil {
		return err
	}

	_, err = r.ensureTaskBelongsToProjectTx(ctx, tx, taskID, projectUUID)
	if err != nil {
		return err
	}

	result, err := tx.Exec(ctx, `
		DELETE FROM task_assignee 
		WHERE task_id = $1 AND user_uuid = $2
	`, taskID, assigneeUUID)
	if err != nil {
		return fmt.Errorf("repository: delete assignee: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrAssigneeNotFound
	}

	return tx.Commit(ctx)
}

func (r *AssigneeRepository) FindByTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	callerUUID uuid.UUID,
) ([]*AssigneeResponse, error) {
	if err := r.ensureUserAccess(ctx, projectUUID, callerUUID); err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT ta.task_id, ta.user_uuid, ta.assigned_at, u.username
		FROM task_assignee ta
		JOIN tasks t ON ta.task_id = t.id
		JOIN project_column pc ON t.column_id = pc.id
		JOIN base_user u ON ta.user_uuid = u.uuid
		WHERE ta.task_id = $1 
		  AND pc.project_uuid = $2
		ORDER BY ta.assigned_at ASC
	`, taskID, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find assignees: %w", err)
	}
	defer rows.Close()

	var assignees []*AssigneeResponse
	for rows.Next() {
		var a AssigneeResponse
		if err := rows.Scan(&a.TaskID, &a.UserUUID, &a.AssignedAt, &a.Username); err != nil {
			return nil, fmt.Errorf("repository: scan assignee: %w", err)
		}
		assignees = append(assignees, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate assignees: %w", err)
	}

	return assignees, nil
}

func (r *AssigneeRepository) ensureUserAccess(ctx context.Context, projectUUID, userUUID uuid.UUID) error {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM project_member WHERE project_uuid = $1 AND user_uuid = $2)
	`, projectUUID, userUUID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("repository: check access: %w", err)
	}
	if !exists {
		return ErrAccessDenied
	}
	return nil
}

func (r *AssigneeRepository) ensureUserAccessTx(ctx context.Context, tx pgx.Tx, projectUUID, userUUID uuid.UUID) error {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM project_member WHERE project_uuid = $1 AND user_uuid = $2)
	`, projectUUID, userUUID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("repository: check access (tx): %w", err)
	}
	if !exists {
		return ErrAccessDenied
	}
	return nil
}

func (r *AssigneeRepository) ensureTaskBelongsToProjectTx(ctx context.Context, tx pgx.Tx, taskID int, projectUUID uuid.UUID) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM tasks t 
			JOIN project_column pc ON t.column_id = pc.id 
			WHERE t.id = $1 AND pc.project_uuid = $2
		)
	`, taskID, projectUUID).Scan(&exists)
	return exists, err
}
