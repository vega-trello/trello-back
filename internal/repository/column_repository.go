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
	color *string,
) (*model.Column, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := r.checkUserAccessTx(ctx, tx, projectUUID, userUUID); err != nil {
		return nil, err
	}

	pos := 0
	if position != nil {
		pos = *position
	} else {
		var maxPos *int
		err = tx.QueryRow(ctx, `
			SELECT MAX(position) FROM project_column 
			WHERE project_uuid = $1
		`, projectUUID).Scan(&maxPos)
		if err != nil {
			return nil, fmt.Errorf("repository: get max position: %w", err)
		}
		if maxPos != nil {
			pos = *maxPos + 1
		} else {
			pos = 0
		}
	}

	var column model.Column
	err = tx.QueryRow(ctx, `
		INSERT INTO project_column (project_uuid, name, position, color, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, project_uuid, name, position, color, created_at
	`, projectUUID, name, pos, color).Scan(
		&column.ID, &column.ProjectUUID, &column.Name, &column.Position, &column.Color, &column.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: create column: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &column, nil
}

func (r *ColumnRepository) FindByProjectUUID(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) ([]*model.Column, error) {
	if _, err := r.checkUserAccess(ctx, projectUUID, userUUID); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, project_uuid, name, position, color, created_at
		FROM project_column
		WHERE project_uuid = $1
		ORDER BY position ASC, created_at ASC
	`, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find columns by project: %w", err)
	}
	defer rows.Close()

	var columns []*model.Column
	for rows.Next() {
		var col model.Column
		if err := rows.Scan(&col.ID, &col.ProjectUUID, &col.Name, &col.Position, &col.Color, &col.CreatedAt); err != nil {
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
		SELECT c.id, c.project_uuid, c.name, c.position, c.color, c.created_at
		FROM project_column c
		JOIN project_member pm ON c.project_uuid = pm.project_uuid
		WHERE c.id = $1 AND pm.user_uuid = $2
	`, columnID, userUUID).Scan(
		&col.ID, &col.ProjectUUID, &col.Name, &col.Position, &col.Color, &col.CreatedAt,
	)
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
	name string,
	position *int,
	color *string,
) (*model.Column, error) {
	var projectUUID uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT project_uuid FROM project_column WHERE id = $1`, columnID).Scan(&projectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrColumnNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find column project: %w", err)
	}
	if _, err := r.checkUserAccess(ctx, projectUUID, userUUID); err != nil {
		return nil, err
	}

	query := "UPDATE project_column SET name = $1"
	args := []interface{}{name}
	argIdx := 2

	if position != nil {
		args = append(args, *position)
		query += fmt.Sprintf(", position = $%d", argIdx)
		argIdx++
	}
	if color != nil {
		args = append(args, *color)
		query += fmt.Sprintf(", color = $%d", argIdx)
		argIdx++
	}

	args = append(args, columnID)
	query += fmt.Sprintf(" WHERE id = $%d RETURNING id, project_uuid, name, position, color, created_at", argIdx)

	var col model.Column
	err = r.db.QueryRow(ctx, query, args...).Scan(&col.ID, &col.ProjectUUID, &col.Name, &col.Position, &col.Color, &col.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrColumnNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: update column: %w", err)
	}
	return &col, nil
}

func (r *ColumnRepository) Delete(
	ctx context.Context,
	columnID int,
	userUUID uuid.UUID,
) error {
	var projectUUID uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT project_uuid FROM project_column WHERE id = $1`, columnID).Scan(&projectUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrColumnNotFound
	}
	if err != nil {
		return fmt.Errorf("repository: find column project: %w", err)
	}
	if _, err := r.checkUserAccess(ctx, projectUUID, userUUID); err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `DELETE FROM project_column WHERE id = $1`, columnID)
	if err != nil {
		return fmt.Errorf("repository: delete column: %w", err)
	}
	return nil
}

func (r *ColumnRepository) Move(
	ctx context.Context,
	columnID int,
	userUUID uuid.UUID,
	direction string,
) (*model.Column, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var col model.Column
	err = tx.QueryRow(ctx, `
		SELECT id, project_uuid, name, position, color, created_at 
		FROM project_column WHERE id = $1
	`, columnID).Scan(&col.ID, &col.ProjectUUID, &col.Name, &col.Position, &col.Color, &col.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrColumnNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find column: %w", err)
	}

	if _, err := r.checkUserAccessTx(ctx, tx, col.ProjectUUID, userUUID); err != nil {
		return nil, err
	}

	var neighborID int
	var neighborPos int

	if direction == "left" {
		err = tx.QueryRow(ctx, `
			SELECT id, position FROM project_column 
			WHERE project_uuid = $1 AND position < $2 AND id != $3
			ORDER BY position DESC 
			LIMIT 1
		`, col.ProjectUUID, col.Position, columnID).Scan(&neighborID, &neighborPos)
	} else {
		err = tx.QueryRow(ctx, `
			SELECT id, position FROM project_column 
			WHERE project_uuid = $1 AND position > $2 AND id != $3
			ORDER BY position ASC 
			LIMIT 1
		`, col.ProjectUUID, col.Position, columnID).Scan(&neighborID, &neighborPos)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return &col, nil
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find neighbor: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE project_column SET position = $1 WHERE id = $2`, neighborPos, columnID)
	if err != nil {
		return nil, fmt.Errorf("repository: update column position: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE project_column SET position = $1 WHERE id = $2`, col.Position, neighborID)
	if err != nil {
		return nil, fmt.Errorf("repository: update neighbor position: %w", err)
	}

	err = tx.QueryRow(ctx, `
		SELECT id, project_uuid, name, position, color, created_at 
		FROM project_column WHERE id = $1
	`, columnID).Scan(&col.ID, &col.ProjectUUID, &col.Name, &col.Position, &col.Color, &col.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("repository: refresh column: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &col, nil
}

func (r *ColumnRepository) checkUserAccess(ctx context.Context, projectUUID, userUUID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_member WHERE project_uuid = $1 AND user_uuid = $2)`, projectUUID, userUUID).Scan(&exists)
	return exists, err
}

func (r *ColumnRepository) checkUserAccessTx(ctx context.Context, tx pgx.Tx, projectUUID, userUUID uuid.UUID) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_member WHERE project_uuid = $1 AND user_uuid = $2)`, projectUUID, userUUID).Scan(&exists)
	return exists, err
}
