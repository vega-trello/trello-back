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

type TaskRepositoryInterface interface {
	FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID) ([]*model.TaskDB, error)
	FindByID(ctx context.Context, taskID int) (*model.TaskDB, error)
	Create(ctx context.Context, input CreateTaskInput) (*model.TaskDB, error)
	Update(ctx context.Context, taskID int, updates TaskUpdates) (*model.TaskDB, error)
	Delete(ctx context.Context, taskID int) error
	Restore(ctx context.Context, taskID int) error
	AddAssignee(ctx context.Context, taskID int, userUUID uuid.UUID) error
	RemoveAssignee(ctx context.Context, taskID int, userUUID uuid.UUID) error
	GetAssignees(ctx context.Context, taskID int) ([]uuid.UUID, error)
}

// структура для создания таски
type CreateTaskInput struct {
	ProjectUUID uuid.UUID
	ColumnID    int
	Title       string
	Description string
	CreatorUUID uuid.UUID
	StatusID    *int // опционально
	StartDate   *time.Time
	EndDate     *time.Time
}

// структура для апдейта таски
type TaskUpdates struct {
	Title       *string
	Description *string
	ColumnID    *int
	StatusID    *int
	StartDate   *time.Time
	EndDate     *time.Time
	ArchivedAt  *time.Time
}

type TaskRepository struct {
	db *pgxpool.Pool
}

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

// FindByProjectUUID возвращает все задачи проекта (кроме удалённых)
// GET /projects/{projectUUID}/tasks
func (r *TaskRepository) FindByProjectUUID(
	ctx context.Context,
	projectUUID uuid.UUID,
) ([]*model.TaskDB, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.column_id, t.status_id, t.creator_uuid,
		       t.title, t.description, t.deleted_at, t.archived_at,
		       t.created_at, t.updated_at, t.start_date, t.end_date
		FROM tasks t
		JOIN project_column pc ON t.column_id = pc.id
		WHERE pc.project_uuid = $1 AND t.deleted_at IS NULL
		ORDER BY t.created_at ASC
	`, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find tasks by project: %w", err)
	}
	defer rows.Close()

	var tasks []*model.TaskDB
	for rows.Next() {
		var task model.TaskDB
		err := rows.Scan(
			&task.ID, &task.ColumnID, &task.StatusID, &task.CreatorUUID,
			&task.Title, &task.Description, &task.DeletedAt, &task.ArchivedAt,
			&task.CreatedAt, &task.UpdatedAt, &task.StartDate, &task.EndDate,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: scan task: %w", err)
		}
		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate tasks: %w", err)
	}

	return tasks, nil
}

// FindByID возвращает задачу по ID (включая удалённые/архивные)
// GET /projects/{projectUUID}/task/{taskID}
func (r *TaskRepository) FindByID(
	ctx context.Context,
	taskID int,
) (*model.TaskDB, error) {
	var task model.TaskDB

	err := r.db.QueryRow(ctx, `
		SELECT id, column_id, status_id, creator_uuid,
		       title, description, deleted_at, archived_at,
		       created_at, updated_at, start_date, end_date
		FROM tasks
		WHERE id = $1
	`, taskID).Scan(
		&task.ID, &task.ColumnID, &task.StatusID, &task.CreatorUUID,
		&task.Title, &task.Description, &task.DeletedAt, &task.ArchivedAt,
		&task.CreatedAt, &task.UpdatedAt, &task.StartDate, &task.EndDate,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find task by id: %w", err)
	}

	return &task, nil
}

// Create создаёт новую задачу
// POST /projects/{projectUUID}/task
func (r *TaskRepository) Create(
	ctx context.Context,
	input CreateTaskInput,
) (*model.TaskDB, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var task model.TaskDB
	now := time.Now()

	// Создаём задачу
	err = tx.QueryRow(ctx, `
		INSERT INTO tasks (
			column_id, status_id, creator_uuid, title, description,
			start_date, end_date, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		RETURNING id, column_id, status_id, creator_uuid, title, description,
		          deleted_at, archived_at, created_at, updated_at, start_date, end_date
	`, input.ColumnID, input.StatusID, input.CreatorUUID, input.Title, input.Description,
		input.StartDate, input.EndDate, now).Scan(
		&task.ID, &task.ColumnID, &task.StatusID, &task.CreatorUUID,
		&task.Title, &task.Description, &task.DeletedAt, &task.ArchivedAt,
		&task.CreatedAt, &task.UpdatedAt, &task.StartDate, &task.EndDate,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: create task: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return &task, nil
}

// Update обновляет задачу (частичное обновление)
// PATCH /projects/{projectUUID}/task/{taskID}
func (r *TaskRepository) Update(
	ctx context.Context,
	taskID int,
	updates TaskUpdates,
) (*model.TaskDB, error) {
	if updates.isEmpty() {
		return r.FindByID(ctx, taskID)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	updatesSQL := "UPDATE tasks SET updated_at = NOW()"
	args := []interface{}{}
	paramIndex := 0 // ← Считаем ТОЛЬКО реальные параметры

	// Title
	if updates.Title != nil {
		paramIndex++
		updatesSQL += fmt.Sprintf(", title = $%d", paramIndex)
		args = append(args, *updates.Title)
	}
	// Description
	if updates.Description != nil {
		paramIndex++
		updatesSQL += fmt.Sprintf(", description = $%d", paramIndex)
		args = append(args, *updates.Description)
	}
	// ColumnID
	if updates.ColumnID != nil {
		paramIndex++
		updatesSQL += fmt.Sprintf(", column_id = $%d", paramIndex)
		args = append(args, *updates.ColumnID)
	}
	// StatusID
	if updates.StatusID != nil {
		paramIndex++
		updatesSQL += fmt.Sprintf(", status_id = $%d", paramIndex)
		args = append(args, *updates.StatusID)
	}
	// StartDate
	if updates.StartDate != nil {
		paramIndex++
		updatesSQL += fmt.Sprintf(", start_date = $%d", paramIndex)
		args = append(args, *updates.StartDate)
	}
	// EndDate
	if updates.EndDate != nil {
		paramIndex++
		updatesSQL += fmt.Sprintf(", end_date = $%d", paramIndex)
		args = append(args, *updates.EndDate)
	}
	// ArchivedAt
	if updates.ArchivedAt != nil {
		if updates.ArchivedAt.IsZero() {
			updatesSQL += ", archived_at = NULL" // ← Без параметра!
		} else {
			paramIndex++
			updatesSQL += fmt.Sprintf(", archived_at = $%d", paramIndex)
			args = append(args, *updates.ArchivedAt)
		}
	}

	// WHERE clause
	paramIndex++
	updatesSQL += fmt.Sprintf(" WHERE id = $%d RETURNING id, column_id, status_id, creator_uuid, title, description, deleted_at, archived_at, created_at, updated_at, start_date, end_date", paramIndex)
	args = append(args, taskID)

	var task model.TaskDB
	err = tx.QueryRow(ctx, updatesSQL, args...).Scan(
		&task.ID, &task.ColumnID, &task.StatusID, &task.CreatorUUID,
		&task.Title, &task.Description, &task.DeletedAt, &task.ArchivedAt,
		&task.CreatedAt, &task.UpdatedAt, &task.StartDate, &task.EndDate,
	)

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

// isEmpty проверяет пустой ли объект обновлений
func (u TaskUpdates) isEmpty() bool {
	return u.Title == nil && u.Description == nil && u.ColumnID == nil &&
		u.StatusID == nil && u.StartDate == nil && u.EndDate == nil && u.ArchivedAt == nil
}

// Delete выполняет soft delete задачи
// DELETE /projects/{projectUUID}/task/{taskID}
func (r *TaskRepository) Delete(ctx context.Context, taskID int) error {
	result, err := r.db.Exec(ctx, `
		UPDATE tasks SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, taskID)
	if err != nil {
		return fmt.Errorf("repository: soft delete task: %w", err)
	}
	if result.RowsAffected() == 0 {
		var exists bool
		err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tasks WHERE id = $1)`, taskID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("repository: check task exists: %w", err)
		}
		if !exists {
			return ErrTaskNotFound
		}
		return ErrTaskDeleted
	}
	return nil
}

// Restore восстанавливает задачу из корзины
// POST /projects/{projectUUID}/task/{taskID}/restore
func (r *TaskRepository) Restore(ctx context.Context, taskID int) error {
	result, err := r.db.Exec(ctx, `
		UPDATE tasks SET deleted_at = NULL
		WHERE id = $1 AND deleted_at IS NOT NULL
	`, taskID)
	if err != nil {
		return fmt.Errorf("repository: restore task: %w", err)
	}
	if result.RowsAffected() == 0 {
		var exists bool
		err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tasks WHERE id = $1)`, taskID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("repository: check task exists: %w", err)
		}
		if !exists {
			return ErrTaskNotFound
		}
		return nil
	}
	return nil
}

// AddAssignee добавляет исполнителя к задаче
// POST /tasks/{taskID}/assignees
func (r *TaskRepository) AddAssignee(
	ctx context.Context,
	taskID int,
	userUUID uuid.UUID,
) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO task_assignee (task_id, user_uuid, assigned_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (task_id, user_uuid) DO NOTHING
	`, taskID, userUUID)
	if err != nil {
		return fmt.Errorf("repository: add assignee: %w", err)
	}
	return nil
}

// RemoveAssignee удаляет исполнителя из задачи
// DELETE /tasks/{taskID}/assignees/{userUUID}
func (r *TaskRepository) RemoveAssignee(
	ctx context.Context,
	taskID int,
	userUUID uuid.UUID,
) error {
	result, err := r.db.Exec(ctx, `
		DELETE FROM task_assignee
		WHERE task_id = $1 AND user_uuid = $2
	`, taskID, userUUID)
	if err != nil {
		return fmt.Errorf("repository: remove assignee: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("repository: assignee not found")
	}
	return nil
}

// GetAssignees возвращает список UUID исполнителей задачи
// GET /tasks/{taskID}/assignees
func (r *TaskRepository) GetAssignees(
	ctx context.Context,
	taskID int,
) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_uuid FROM task_assignee WHERE task_id = $1
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("repository: get assignees: %w", err)
	}
	defer rows.Close()

	var assignees []uuid.UUID
	for rows.Next() {
		var userUUID uuid.UUID
		err := rows.Scan(&userUUID)
		if err != nil {
			return nil, fmt.Errorf("repository: scan assignee: %w", err)
		}
		assignees = append(assignees, userUUID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate assignees: %w", err)
	}

	return assignees, nil
}
