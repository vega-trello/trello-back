package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vega-trello/trello-back/internal/model"
)

// AssignUserInput отражает тело запроса POST /projects/{projectUUID}/assignee
type AssignUserInput struct {
	UserUUID uuid.UUID `json:"user_uuid"`
}

type AssigneeRepositoryInterface interface {
	FindByTask(ctx context.Context, projectUUID uuid.UUID, taskID int) ([]*model.User, error)
	AddToTask(ctx context.Context, projectUUID uuid.UUID, taskID int, input AssignUserInput) error
	RemoveFromTask(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) error
}

type AssigneeRepository struct {
	db *pgxpool.Pool
}

func NewAssigneeRepository(db *pgxpool.Pool) *AssigneeRepository {
	return &AssigneeRepository{db: db}
}

// FindByTask возвращает всех исполнителей задачи в рамках проекта
func (r *AssigneeRepository) FindByTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
) ([]*model.User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT bu.uuid, bu.username, bu.user_type, bu.created_at, bu.updated_at
		FROM task_assignee ta
		JOIN tasks t ON ta.task_id = t.id
		JOIN project_column pc ON t.column_id = pc.id
		JOIN base_user bu ON ta.user_uuid = bu.uuid
		WHERE pc.project_uuid = $1 AND t.id = $2
		ORDER BY ta.assigned_at ASC
	`, projectUUID, taskID)
	if err != nil {
		return nil, fmt.Errorf("repository: find assignees by task: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.UUID, &user.Username, &user.UserType, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan assignee user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate assignees: %w", err)
	}

	return users, nil
}

// AddToTask назначает пользователя на задачу
func (r *AssigneeRepository) AddToTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	input AssignUserInput,
) error {
	var taskInProject bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM tasks t
			JOIN project_column pc ON t.column_id = pc.id
			WHERE t.id = $1 AND pc.project_uuid = $2
		)
	`, taskID, projectUUID).Scan(&taskInProject)
	if err != nil {
		return fmt.Errorf("repository: verify task project: %w", err)
	}
	if !taskInProject {
		return ErrTaskNotFoundAssgnee
	}

	// Вставляем назначение. ON CONFLICT делает операцию идемпотентной
	_, err = r.db.Exec(ctx, `
		INSERT INTO task_assignee (task_id, user_uuid, assigned_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (task_id, user_uuid) DO NOTHING
	`, taskID, input.UserUUID)

	if err != nil {
		// Если вдруг конфликт всё же пробьётся (race condition)
		if IsUniqueViolation(err) {
			return ErrAssigneeAlreadyExists
		}
		return fmt.Errorf("repository: insert assignee: %w", err)
	}

	return nil
}

// RemoveFromTask убирает назначение пользователя с задачи
func (r *AssigneeRepository) RemoveFromTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
) error {
	result, err := r.db.Exec(ctx, `
		DELETE FROM task_assignee ta
		USING tasks t
		JOIN project_column pc ON t.column_id = pc.id
		WHERE ta.task_id = $1
		  AND ta.user_uuid = $2
		  AND t.id = $1
		  AND pc.project_uuid = $3
		RETURNING ta.task_id
	`, taskID, userUUID, projectUUID)
	if err != nil {
		return fmt.Errorf("repository: remove assignee: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAssigneeNotFound
	}

	return nil
}
