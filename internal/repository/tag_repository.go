package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vega-trello/trello-back/internal/model"
)

var ErrTagAlreadyExists = errors.New("tag already exists in project")

type TagRepository struct {
	db *pgxpool.Pool
}

func NewTagRepository(db *pgxpool.Pool) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) Create(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	name string,
	color string,
) (*model.Tag, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.ensureUserAccessTx(ctx, tx, projectUUID, userUUID); err != nil {
		return nil, err
	}

	var tag model.Tag
	err = tx.QueryRow(ctx, `
		INSERT INTO tag (project_uuid, name, color, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, project_uuid, name, color, created_at, updated_at
	`, projectUUID, name, color).Scan(
		&tag.ID, &tag.ProjectUUID, &tag.Name, &tag.Color, &tag.CreatedAt, &tag.UpdatedAt,
	)
	if err != nil {
		if pgErr := (*pgconn.PgError)(nil); errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrTagAlreadyExists
		}
		return nil, fmt.Errorf("repository: create tag: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &tag, nil
}

func (r *TagRepository) FindByProjectUUID(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) ([]*model.Tag, error) {
	if err := r.ensureUserAccess(ctx, projectUUID, userUUID); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, project_uuid, name, color, created_at, updated_at
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
		if err := rows.Scan(&tag.ID, &tag.ProjectUUID, &tag.Name, &tag.Color, &tag.CreatedAt, &tag.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan tag: %w", err)
		}
		tags = append(tags, &tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate tags: %w", err)
	}
	return tags, nil
}

func (r *TagRepository) FindByTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
) ([]*model.Tag, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id, t.project_uuid, t.name, t.color, t.created_at, t.updated_at
		FROM tag t
		JOIN task_tag tt ON t.id = tt.tag_id
		JOIN tasks ta ON tt.task_id = ta.id
		JOIN project_column pc ON ta.column_id = pc.id
		JOIN project_member pm ON pc.project_uuid = pm.project_uuid
		WHERE tt.task_id = $1 
		  AND pc.project_uuid = $2 
		  AND pm.user_uuid = $3     
		ORDER BY t.name ASC
	`, taskID, projectUUID, userUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find tags by task: %w", err)
	}
	defer rows.Close()

	var tags []*model.Tag
	for rows.Next() {
		var tag model.Tag
		if err := rows.Scan(&tag.ID, &tag.ProjectUUID, &tag.Name, &tag.Color, &tag.CreatedAt, &tag.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan tag: %w", err)
		}
		tags = append(tags, &tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate tags: %w", err)
	}
	return tags, nil
}

func (r *TagRepository) Update(
	ctx context.Context,
	tagID int,
	userUUID uuid.UUID,
	name string,
	color string,
) (*model.Tag, error) {
	var projectUUID uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT project_uuid FROM tag WHERE id = $1`, tagID).Scan(&projectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTagNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find tag project: %w", err)
	}
	if err := r.ensureUserAccess(ctx, projectUUID, userUUID); err != nil {
		return nil, err
	}

	var tag model.Tag
	err = r.db.QueryRow(ctx, `
		UPDATE tag
		SET name = $1, color = $2, updated_at = NOW() 
		WHERE id = $3
		RETURNING id, project_uuid, name, color, created_at, updated_at
	`, name, color, tagID).Scan(
		&tag.ID, &tag.ProjectUUID, &tag.Name, &tag.Color, &tag.CreatedAt, &tag.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTagNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: update tag: %w", err)
	}
	return &tag, nil
}

func (r *TagRepository) Delete(
	ctx context.Context,
	tagID int,
	userUUID uuid.UUID,
) error {
	var projectUUID uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT project_uuid FROM tag WHERE id = $1`, tagID).Scan(&projectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrTagNotFound
	}
	if err != nil {
		return fmt.Errorf("repository: find tag project: %w", err)
	}
	if err := r.ensureUserAccess(ctx, projectUUID, userUUID); err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `DELETE FROM tag WHERE id = $1`, tagID)
	if err != nil {
		return fmt.Errorf("repository: delete tag: %w", err)
	}
	return nil
}

func (r *TagRepository) AddToTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	taskID int,
	tagID int,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.ensureUserAccessTx(ctx, tx, projectUUID, userUUID); err != nil {
		return err
	}

	var tagProjectUUID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT project_uuid FROM tag WHERE id = $1`, tagID).Scan(&tagProjectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrTagNotFound
	}
	if err != nil {
		return fmt.Errorf("repository: check tag project: %w", err)
	}
	if tagProjectUUID != projectUUID {
		return errors.New("tag does not belong to this project")
	}

	var taskProjectUUID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT pc.project_uuid FROM tasks t JOIN project_column pc ON t.column_id = pc.id WHERE t.id = $1`, taskID).Scan(&taskProjectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrTaskNotFound
	}
	if err != nil {
		return fmt.Errorf("repository: check task project: %w", err)
	}
	if taskProjectUUID != projectUUID {
		return errors.New("task does not belong to this project")
	}
	_, err = tx.Exec(ctx, `INSERT INTO task_tag (task_id, tag_id, added_at) VALUES ($1, $2, NOW()) ON CONFLICT (task_id, tag_id) DO NOTHING`, taskID, tagID)
	if err != nil {
		return fmt.Errorf("repository: assign tag to task: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *TagRepository) RemoveFromTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	taskID int,
	tagID int,
) error {
	if err := r.ensureUserAccess(ctx, projectUUID, userUUID); err != nil {
		return err
	}
	result, err := r.db.Exec(ctx, `DELETE FROM task_tag WHERE task_id = $1 AND tag_id = $2`, taskID, tagID)
	if err != nil {
		return fmt.Errorf("repository: remove tag from task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTagNotFound
	}
	return nil
}

func (r *TagRepository) ensureUserAccess(ctx context.Context, projectUUID, userUUID uuid.UUID) error {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_member WHERE project_uuid = $1 AND user_uuid = $2)`, projectUUID, userUUID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("repository: check access: %w", err)
	}
	if !exists {
		return ErrAccessDenied
	}
	return nil
}

func (r *TagRepository) ensureUserAccessTx(ctx context.Context, tx pgx.Tx, projectUUID, userUUID uuid.UUID) error {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_member WHERE project_uuid = $1 AND user_uuid = $2)`, projectUUID, userUUID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("repository: check access (tx): %w", err)
	}
	if !exists {
		return ErrAccessDenied
	}
	return nil
}
