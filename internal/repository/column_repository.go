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

type ColumnRepositoryInterface interface {
	Create(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, name string, position *int) (*model.Column, error)
	FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Column, error)
	FindByID(ctx context.Context, columnID int, userUUID uuid.UUID) (*model.Column, error)
	Update(ctx context.Context, columnID int, userUUID uuid.UUID, name *string, position *int) (*model.Column, error)
	Delete(ctx context.Context, columnID int, userUUID uuid.UUID) error
	Move(ctx context.Context, columnID int, userUUID uuid.UUID, newPosition int) (*model.Column, error)
}

type ColumnRepository struct {
	db *pgxpool.Pool
}

func NewColumnRepository(db *pgxpool.Pool) *ColumnRepository {
	return &ColumnRepository{db: db}
}

func (r *ColumnRepository) Create(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	name string,
	position *int,
) (*model.Column, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.ensureUserAccessTx(ctx, tx, projectUUID, userUUID); err != nil {
		return nil, err
	}

	pos := position
	if pos == nil {
		var maxPos int
		err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(position), -1) FROM project_column WHERE project_uuid = $1`, projectUUID).Scan(&maxPos)
		if err != nil {
			return nil, fmt.Errorf("repository: calculate max position: %w", err)
		}
		p := maxPos + 1
		pos = &p
	}

	var col model.Column
	err = tx.QueryRow(ctx, `
		INSERT INTO project_column (project_uuid, name, position, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, project_uuid, name, position, created_at
	`, projectUUID, name, *pos).Scan(&col.ID, &col.ProjectUUID, &col.Name, &col.Position, &col.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("repository: create column: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &col, nil
}

func (r *ColumnRepository) FindByProjectUUID(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) ([]*model.Column, error) {
	if err := r.ensureUserAccess(ctx, projectUUID, userUUID); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, project_uuid, name, position, created_at
		FROM project_column
		WHERE project_uuid = $1
		ORDER BY position ASC
	`, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find columns by project: %w", err)
	}
	defer rows.Close()

	var columns []*model.Column
	for rows.Next() {
		var col model.Column
		if err := rows.Scan(&col.ID, &col.ProjectUUID, &col.Name, &col.Position, &col.CreatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan column: %w", err)
		}
		columns = append(columns, &col)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate columns: %w", err)
	}
	return columns, nil
}

func (r *ColumnRepository) FindByID(
	ctx context.Context,
	columnID int,
	userUUID uuid.UUID,
) (*model.Column, error) {
	var col model.Column
	err := r.db.QueryRow(ctx, `
		SELECT pc.id, pc.project_uuid, pc.name, pc.position, pc.created_at
		FROM project_column pc
		JOIN project_member pm ON pc.project_uuid = pm.project_uuid
		WHERE pc.id = $1 AND pm.user_uuid = $2
	`, columnID, userUUID).Scan(&col.ID, &col.ProjectUUID, &col.Name, &col.Position, &col.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrColumnNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find column by id: %w", err)
	}
	return &col, nil
}

func (r *ColumnRepository) Update(
	ctx context.Context,
	columnID int,
	userUUID uuid.UUID,
	name *string,
	position *int,
) (*model.Column, error) {
	existing, err := r.FindByID(ctx, columnID, userUUID)
	if err != nil {
		return nil, err
	}
	if err := r.ensureUserAccess(ctx, existing.ProjectUUID, userUUID); err != nil {
		return nil, err
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := "UPDATE project_column SET "
	args := []interface{}{}
	argIdx := 1
	if name != nil {
		args = append(args, *name)
		query += fmt.Sprintf("name = $%d", argIdx)
		argIdx++
	}
	if position != nil {
		if argIdx > 1 {
			query += ", "
		}
		args = append(args, *position)
		query += fmt.Sprintf("position = $%d", argIdx)
		argIdx++
	}
	if argIdx == 1 {
		return existing, nil
	}
	args = append(args, columnID)
	query += fmt.Sprintf(" WHERE id = $%d RETURNING id, project_uuid, name, position, created_at", argIdx)

	var updated model.Column
	err = tx.QueryRow(ctx, query, args...).Scan(&updated.ID, &updated.ProjectUUID, &updated.Name, &updated.Position, &updated.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrColumnNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: update column: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &updated, nil
}

func (r *ColumnRepository) Delete(
	ctx context.Context,
	columnID int,
	userUUID uuid.UUID,
) error {
	existing, err := r.FindByID(ctx, columnID, userUUID)
	if err != nil {
		return err
	}
	if err := r.ensureUserAccess(ctx, existing.ProjectUUID, userUUID); err != nil {
		return err
	}
	result, err := r.db.Exec(ctx, `DELETE FROM project_column WHERE id = $1`, columnID)
	if err != nil {
		return fmt.Errorf("repository: delete column: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrColumnNotFound
	}
	return nil
}

func (r *ColumnRepository) Move(
	ctx context.Context,
	columnID int,
	userUUID uuid.UUID,
	newPosition int,
) (*model.Column, error) {
	if newPosition < 0 {
		return nil, ErrInvalidPosition
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id, project_uuid, name, position, created_at
		FROM project_column
		WHERE project_uuid = (SELECT project_uuid FROM project_column WHERE id = $1)
		ORDER BY position ASC
	`, columnID)
	if err != nil {
		return nil, fmt.Errorf("repository: fetch project columns: %w", err)
	}
	defer rows.Close()

	var allCols []*model.Column
	var movingCol *model.Column
	for rows.Next() {
		var c model.Column
		if err := rows.Scan(&c.ID, &c.ProjectUUID, &c.Name, &c.Position, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan column: %w", err)
		}
		if c.ID == columnID {
			movingCol = &c
		}
		allCols = append(allCols, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate columns: %w", err)
	}
	if movingCol == nil {
		return nil, ErrColumnNotFound
	}
	if err := r.ensureUserAccessTx(ctx, tx, movingCol.ProjectUUID, userUUID); err != nil {
		return nil, err
	}

	filtered := make([]*model.Column, 0, len(allCols)-1)
	for _, c := range allCols {
		if c.ID != columnID {
			filtered = append(filtered, c)
		}
	}
	if newPosition >= len(filtered) {
		newPosition = len(filtered)
	}
	finalOrder := append(filtered[:newPosition], append([]*model.Column{movingCol}, filtered[newPosition:]...)...)

	for i, c := range finalOrder {
		_, err = tx.Exec(ctx, `UPDATE project_column SET position = $1 WHERE id = $2`, i, c.ID)
		if err != nil {
			return nil, fmt.Errorf("repository: update column position: %w", err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	movingCol.Position = newPosition
	return movingCol, nil
}

func (r *ColumnRepository) ensureUserAccess(ctx context.Context, projectUUID, userUUID uuid.UUID) error {
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

func (r *ColumnRepository) ensureUserAccessTx(ctx context.Context, tx pgx.Tx, projectUUID, userUUID uuid.UUID) error {
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
