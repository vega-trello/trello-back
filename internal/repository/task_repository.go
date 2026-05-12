package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vega-trello/trello-back/internal/model"
)

type TaskRepository struct {
	db *pgxpool.Pool
}

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(
	ctx context.Context,
	projectUUID uuid.UUID,
	columnID int,
	creatorUUID uuid.UUID,
	title string,
	description string,
	startDate *time.Time,
	endDate *time.Time,
) (*model.TaskDB, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var colProjectUUID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT project_uuid FROM project_column WHERE id = $1`, columnID).Scan(&colProjectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidColumn
	}
	if err != nil {
		return nil, fmt.Errorf("repository: check column: %w", err)
	}
	if colProjectUUID != projectUUID {
		return nil, ErrInvalidColumn
	}
	if _, err := r.checkUserAccessTx(ctx, tx, projectUUID, creatorUUID); err != nil {
		return nil, err
	}

	var task model.TaskDB
	err = tx.QueryRow(ctx, `
		INSERT INTO tasks (column_id, creator_uuid, title, description, start_date, end_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, column_id, status_id, creator_uuid, title, description, archived_at, created_at, updated_at, start_date, end_date
	`, columnID, creatorUUID, title, description, startDate, endDate, time.Now()).Scan(
		&task.ID, &task.ColumnID, &task.StatusID, &task.CreatorUUID, &task.Title, &task.Description,
		&task.ArchivedAt, &task.CreatedAt, &task.UpdatedAt, &task.StartDate, &task.EndDate,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: create task: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &task, nil
}

func (r *TaskRepository) FindByID(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
) (*model.TaskDB, error) {
	var task model.TaskDB
	err := r.db.QueryRow(ctx, `
		SELECT t.id, t.column_id, t.status_id, t.creator_uuid, t.title, t.description,
		       t.archived_at, t.created_at, t.updated_at, t.start_date, t.end_date
		FROM tasks t
		JOIN project_column pc ON t.column_id = pc.id
		JOIN project_member pm ON pc.project_uuid = pm.project_uuid
		WHERE t.id = $1 AND pc.project_uuid = $2 AND pm.user_uuid = $3
	`, taskID, projectUUID, userUUID).Scan(
		&task.ID, &task.ColumnID, &task.StatusID, &task.CreatorUUID, &task.Title, &task.Description,
		&task.ArchivedAt, &task.CreatedAt, &task.UpdatedAt, &task.StartDate, &task.EndDate,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find task by id: %w", err)
	}
	return &task, nil
}

func (r *TaskRepository) FindByProjectUUID(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	archived *bool,
) ([]*model.TaskDB, error) {
	if _, err := r.checkUserAccess(ctx, projectUUID, userUUID); err != nil {
		return nil, err
	}
	query := `
		SELECT t.id, t.column_id, t.status_id, t.creator_uuid, t.title, t.description,
		       t.archived_at, t.created_at, t.updated_at, t.start_date, t.end_date
		FROM tasks t
		JOIN project_column pc ON t.column_id = pc.id
		WHERE pc.project_uuid = $1
	`
	args := []interface{}{projectUUID}
	if archived != nil {
		if *archived {
			query += " AND t.archived_at IS NOT NULL"
		} else {
			query += " AND t.archived_at IS NULL"
		}
	}
	query += " ORDER BY t.created_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: find tasks by project: %w", err)
	}
	defer rows.Close()

	var tasks []*model.TaskDB
	for rows.Next() {
		var task model.TaskDB
		if err := rows.Scan(&task.ID, &task.ColumnID, &task.StatusID, &task.CreatorUUID, &task.Title, &task.Description,
			&task.ArchivedAt, &task.CreatedAt, &task.UpdatedAt, &task.StartDate, &task.EndDate); err != nil {
			return nil, fmt.Errorf("repository: scan task: %w", err)
		}
		tasks = append(tasks, &task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate tasks: %w", err)
	}
	return tasks, nil
}

func (r *TaskRepository) FindByColumn(
	ctx context.Context,
	columnID int,
	userUUID uuid.UUID,
) ([]*model.TaskDB, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.column_id, t.status_id, t.creator_uuid, t.title, t.description,
		       t.archived_at, t.created_at, t.updated_at, t.start_date, t.end_date
		FROM tasks t
		JOIN project_column pc ON t.column_id = pc.id
		JOIN project_member pm ON pc.project_uuid = pm.project_uuid
		WHERE t.column_id = $1 AND pm.user_uuid = $2
		ORDER BY t.created_at DESC
	`, columnID, userUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find tasks by column: %w", err)
	}
	defer rows.Close()

	var tasks []*model.TaskDB
	for rows.Next() {
		var task model.TaskDB
		if err := rows.Scan(&task.ID, &task.ColumnID, &task.StatusID, &task.CreatorUUID, &task.Title, &task.Description,
			&task.ArchivedAt, &task.CreatedAt, &task.UpdatedAt, &task.StartDate, &task.EndDate); err != nil {
			return nil, fmt.Errorf("repository: scan task: %w", err)
		}
		tasks = append(tasks, &task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate tasks: %w", err)
	}
	return tasks, nil
}

func (r *TaskRepository) Update(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
	title *string,
	description *string,
	startDate **time.Time,
	endDate **time.Time,
	columnID *int,
	archived *bool,
) (*model.TaskDB, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := r.checkUserAccessTx(ctx, tx, projectUUID, userUUID); err != nil {
		return nil, err
	}

	query := "UPDATE tasks SET updated_at = NOW()"
	args := []interface{}{}
	argIdx := 1

	if title != nil {
		args = append(args, *title)
		query += fmt.Sprintf(", title = $%d", argIdx)
		argIdx++
	}
	if description != nil {
		args = append(args, *description)
		query += fmt.Sprintf(", description = $%d", argIdx)
		argIdx++
	}
	if startDate != nil && *startDate != nil {
		args = append(args, **startDate)
		query += fmt.Sprintf(", start_date = $%d", argIdx)
		argIdx++
	} else if startDate != nil {
		query += ", start_date = NULL"
	}
	if endDate != nil && *endDate != nil {
		args = append(args, **endDate)
		query += fmt.Sprintf(", end_date = $%d", argIdx)
		argIdx++
	} else if endDate != nil {
		query += ", end_date = NULL"
	}
	if columnID != nil {
		var targetProjectUUID uuid.UUID
		err = tx.QueryRow(ctx, `SELECT project_uuid FROM project_column WHERE id = $1`, *columnID).Scan(&targetProjectUUID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidColumn
		}
		if err != nil {
			return nil, fmt.Errorf("repository: check target column: %w", err)
		}
		if targetProjectUUID != projectUUID {
			return nil, ErrInvalidColumn
		}
		args = append(args, *columnID)
		query += fmt.Sprintf(", column_id = $%d", argIdx)
		argIdx++
	}
	if archived != nil {
		if *archived {
			query += ", archived_at = NOW()"
		} else {
			query += ", archived_at = NULL"
		}
	}

	args = append(args, taskID, projectUUID)
	query += fmt.Sprintf(" WHERE id = $%d AND column_id IN (SELECT id FROM project_column WHERE project_uuid = $%d)", argIdx, argIdx+1)
	query += " RETURNING id, column_id, status_id, creator_uuid, title, description, archived_at, created_at, updated_at, start_date, end_date"

	var task model.TaskDB
	err = tx.QueryRow(ctx, query, args...).Scan(&task.ID, &task.ColumnID, &task.StatusID, &task.CreatorUUID, &task.Title, &task.Description,
		&task.ArchivedAt, &task.CreatedAt, &task.UpdatedAt, &task.StartDate, &task.EndDate)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: update task: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &task, nil
}

func (r *TaskRepository) Delete(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := r.checkUserAccessTx(ctx, tx, projectUUID, userUUID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		DELETE FROM tasks
		WHERE id = $1 AND column_id IN (SELECT id FROM project_column WHERE project_uuid = $2)
	`, taskID, projectUUID)
	if err != nil {
		return fmt.Errorf("repository: delete task: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *TaskRepository) Move(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	targetColumnID int,
	userUUID uuid.UUID,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := r.checkUserAccessTx(ctx, tx, projectUUID, userUUID); err != nil {
		return err
	}
	var targetProjectUUID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT project_uuid FROM project_column WHERE id = $1`, targetColumnID).Scan(&targetProjectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidColumn
	}
	if err != nil {
		return fmt.Errorf("repository: check target column: %w", err)
	}
	if targetProjectUUID != projectUUID {
		return ErrInvalidColumn
	}
	_, err = tx.Exec(ctx, `
		UPDATE tasks SET column_id = $1, updated_at = NOW()
		WHERE id = $2 AND column_id IN (SELECT id FROM project_column WHERE project_uuid = $3)
	`, targetColumnID, taskID, projectUUID)
	if err != nil {
		return fmt.Errorf("repository: move task: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *TaskRepository) Archive(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
	archive bool,
) error {
	if _, err := r.checkUserAccess(ctx, projectUUID, userUUID); err != nil {
		return err
	}
	query := `
		UPDATE tasks
		SET archived_at = CASE WHEN $1 THEN NOW() ELSE NULL END, updated_at = NOW()
		WHERE id = $2 AND column_id IN (SELECT id FROM project_column WHERE project_uuid = $3)
	`
	_, err := r.db.Exec(ctx, query, archive, taskID, projectUUID)
	if err != nil {
		return fmt.Errorf("repository: archive task: %w", err)
	}
	return nil
}

func (r *TaskRepository) checkUserAccess(ctx context.Context, projectUUID, userUUID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_member WHERE project_uuid = $1 AND user_uuid = $2)`, projectUUID, userUUID).Scan(&exists)
	return exists, err
}

func (r *TaskRepository) checkUserAccessTx(ctx context.Context, tx pgx.Tx, projectUUID, userUUID uuid.UUID) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_member WHERE project_uuid = $1 AND user_uuid = $2)`, projectUUID, userUUID).Scan(&exists)
	return exists, err
}
