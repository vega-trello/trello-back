package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vega-trello/trello-back/internal/model"
)

var (
	ErrTagNotFound      = errors.New("tag not found")
	ErrTagAlreadyExists = errors.New("tag with this name already exists in project")
)

// TagUpdates отражает тело запроса PATCH /projects/{projectUUID}/tag
// Поля с указателями: nil = не менять
type TagUpdates struct {
	Name  *string `json:"name,omitempty"`
	Color *int    `json:"color,omitempty"`
}

// AssignTagInput отражает тело запроса POST /projects/{projectUUID}/task/tags
type AssignTagInput struct {
	TaskID int `json:"task_id"`
	TagID  int `json:"tag_id"`
}

type TagRepositoryInterface interface {
	FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID) ([]*model.Tag, error)
	Create(ctx context.Context, projectUUID uuid.UUID, name string, color int) (*model.Tag, error)
	Update(ctx context.Context, projectUUID uuid.UUID, tagID int, updates TagUpdates) (*model.Tag, error)
	Delete(ctx context.Context, projectUUID uuid.UUID, tagID int) error
	FindByTask(ctx context.Context, projectUUID uuid.UUID, taskID int) ([]*model.Tag, error)
	AddToTask(ctx context.Context, projectUUID uuid.UUID, input AssignTagInput) error
	RemoveFromTask(ctx context.Context, projectUUID uuid.UUID, taskID int, tagID int) error
}

type TagRepository struct {
	db *pgxpool.Pool
}

func NewTagRepository(db *pgxpool.Pool) *TagRepository {
	return &TagRepository{db: db}
}

// FindByProjectUUID возвращает все теги проекта
func (r *TagRepository) FindByProjectUUID(
	ctx context.Context,
	projectUUID uuid.UUID,
) ([]*model.Tag, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, project_uuid, name, color, created_at
		FROM tag
		WHERE project_uuid = $1
		ORDER BY name ASC
	`, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find tags by project: %w", err)
	}
	defer rows.Close()

	var tags []*model.Tag
	for rows.Next() {
		var tag model.Tag
		if err := rows.Scan(&tag.ID, &tag.ProjectUUID, &tag.Name, &tag.Color, &tag.CreatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan tag: %w", err)
		}
		tags = append(tags, &tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate tags: %w", err)
	}

	return tags, nil
}

// Create создаёт новый тег в проекте
func (r *TagRepository) Create(
	ctx context.Context,
	projectUUID uuid.UUID,
	name string,
	color int,
) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.QueryRow(ctx, `
		INSERT INTO tag (project_uuid, name, color)
		VALUES ($1, $2, $3)
		RETURNING id, project_uuid, name, color, created_at
	`, projectUUID, name, color).Scan(
		&tag.ID, &tag.ProjectUUID, &tag.Name, &tag.Color, &tag.CreatedAt,
	)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrTagAlreadyExists
		}
		return nil, fmt.Errorf("repository: create tag: %w", err)
	}
	return &tag, nil
}

func (r *TagRepository) Update(
	ctx context.Context,
	projectUUID uuid.UUID,
	tagID int,
	updates TagUpdates,
) (*model.Tag, error) {
	if updates.isEmpty() {
		return r.findByID(ctx, projectUUID, tagID)
	}

	exists, err := r.tagExistsInProject(ctx, projectUUID, tagID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrTagNotFound
	}

	var setParts []string
	var args []interface{}
	idx := 1

	if updates.Name != nil {
		setParts = append(setParts, "name = $"+strconv.Itoa(idx))
		args = append(args, *updates.Name)
		idx++
	}
	if updates.Color != nil {
		setParts = append(setParts, "color = $"+strconv.Itoa(idx))
		args = append(args, *updates.Color)
		idx++
	}

	setClause := strings.Join(setParts, ", ")
	query := "UPDATE tag SET " + setClause + //тут может быть ошибка, но это не ошибка, а предупреждение (код компелится)
		" WHERE id = $" + strconv.Itoa(idx) +
		" AND project_uuid = $" + strconv.Itoa(idx+1) +
		" RETURNING id, project_uuid, name, color, created_at"

	args = append(args, tagID, projectUUID)

	var tag model.Tag
	err = r.db.QueryRow(ctx, query, args...).Scan(
		&tag.ID, &tag.ProjectUUID, &tag.Name, &tag.Color, &tag.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTagNotFound
	}
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrTagAlreadyExists
		}
		return nil, fmt.Errorf("repository: update tag: %w", err)
	}
	return &tag, nil
}

func (u TagUpdates) isEmpty() bool {
	return u.Name == nil && u.Color == nil
}

// findByID возвращает тег по ID в рамках проекта (внутренний хелпер)
func (r *TagRepository) findByID(
	ctx context.Context,
	projectUUID uuid.UUID,
	tagID int,
) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.QueryRow(ctx, `
		SELECT id, project_uuid, name, color, created_at
		FROM tag
		WHERE id = $1 AND project_uuid = $2
	`, tagID, projectUUID).Scan(
		&tag.ID, &tag.ProjectUUID, &tag.Name, &tag.Color, &tag.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTagNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find tag by id: %w", err)
	}
	return &tag, nil
}

// tagExistsInProject проверяет, существует ли тег в проекте
func (r *TagRepository) tagExistsInProject(
	ctx context.Context,
	projectUUID uuid.UUID,
	tagID int,
) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM tag WHERE id = $1 AND project_uuid = $2
		)
	`, tagID, projectUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("repository: check tag exists: %w", err)
	}
	return exists, nil
}

// Delete удаляет тег из проекта
// ON DELETE CASCADE в task_tag автоматически удалит все связи с задачами
func (r *TagRepository) Delete(
	ctx context.Context,
	projectUUID uuid.UUID,
	tagID int,
) error {
	result, err := r.db.Exec(ctx, `
		DELETE FROM tag
		WHERE id = $1 AND project_uuid = $2
	`, tagID, projectUUID)
	if err != nil {
		return fmt.Errorf("repository: delete tag: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTagNotFound
	}
	return nil
}

// FindByTask возвращает все теги, назначенные задаче
// Проверяет, что задача принадлежит проекту (защита от IDOR)
func (r *TagRepository) FindByTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
) ([]*model.Tag, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.project_uuid, t.name, t.color, t.created_at
		FROM task_tag tt
		JOIN tag t ON tt.tag_id = t.id
		JOIN tasks ts ON tt.task_id = ts.id
		JOIN project_column pc ON ts.column_id = pc.id
		WHERE pc.project_uuid = $1 AND ts.id = $2
		ORDER BY t.name ASC
	`, projectUUID, taskID)
	if err != nil {
		return nil, fmt.Errorf("repository: find tags by task: %w", err)
	}
	defer rows.Close()

	var tags []*model.Tag
	for rows.Next() {
		var tag model.Tag
		if err := rows.Scan(&tag.ID, &tag.ProjectUUID, &tag.Name, &tag.Color, &tag.CreatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan tag: %w", err)
		}
		tags = append(tags, &tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate tags: %w", err)
	}

	return tags, nil
}

// AddToTask назначает тег задаче
func (r *TagRepository) AddToTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	input AssignTagInput,
) error {
	taskInProject, err := r.taskInProject(ctx, projectUUID, input.TaskID)
	if err != nil {
		return err
	}
	if !taskInProject {
		return ErrTaskNotFound
	}

	tagInProject, err := r.tagExistsInProject(ctx, projectUUID, input.TagID)
	if err != nil {
		return err
	}
	if !tagInProject {
		return ErrTagNotFound
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO task_tag (task_id, tag_id, added_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (task_id, tag_id) DO NOTHING
	`, input.TaskID, input.TagID)
	if err != nil {
		return fmt.Errorf("repository: assign tag to task: %w", err)
	}
	return nil
}

// RemoveFromTask убирает тег от задачи
func (r *TagRepository) RemoveFromTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	tagID int,
) error {
	// Удаляем только если задача принадлежит проекту (безопасность)
	result, err := r.db.Exec(ctx, `
		DELETE FROM task_tag tt
		USING tasks t
		JOIN project_column pc ON t.column_id = pc.id
		WHERE tt.task_id = $1
		  AND tt.tag_id = $2
		  AND t.id = $1
		  AND pc.project_uuid = $3
	`, taskID, tagID, projectUUID)
	if err != nil {
		return fmt.Errorf("repository: remove tag from task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("repository: assignment not found")
	}
	return nil
}

// taskInProject проверяет, что задача принадлежит проекту
func (r *TagRepository) taskInProject(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM tasks t
			JOIN project_column pc ON t.column_id = pc.id
			WHERE t.id = $1 AND pc.project_uuid = $2
		)
	`, taskID, projectUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("repository: check task project: %w", err)
	}
	return exists, nil
}
