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

var ErrColumnNotFound = errors.New("column not found")

type ColumnRepositoryInterface interface {
	Create(ctx context.Context, projectUUID uuid.UUID, name string, position int) (*model.Column, error)
	FindByID(ctx context.Context, id int) (*model.Column, error)
	FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID) ([]*model.Column, error)
	Update(ctx context.Context, id int, name *string) (*model.Column, error)
	Delete(ctx context.Context, id int) error
}

type ColumnRepository struct {
	db *pgxpool.Pool
}

func NewColumnRepository(db *pgxpool.Pool) *ColumnRepository {
	return &ColumnRepository{db: db}
}

// POST /projects/{projectUUID}/columns
func (r *ColumnRepository) Create(
	ctx context.Context,
	projectUUID uuid.UUID,
	name string,
	position int,
) (*model.Column, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var column model.Column
	err = tx.QueryRow(ctx, `
		INSERT INTO project_column (project_uuid, name, position, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, project_uuid, name, position, created_at
	`, projectUUID, name, position).Scan(
		&column.ID,
		&column.ProjectUUID,
		&column.Name,
		&column.Position,
		&column.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: create column: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return &column, nil
}

// GET /projects/{projectUUID}/columns
func (r *ColumnRepository) FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID) ([]*model.Column, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, project_uuid, name, position, created_at
		FROM project_column
		WHERE project_uuid = $1
		ORDER BY position ASC, created_at ASC
	`, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find columns by project uuid: %w", err)
	}
	defer rows.Close()

	var columns []*model.Column
	for rows.Next() {
		var column model.Column
		err := rows.Scan(
			&column.ID,
			&column.ProjectUUID,
			&column.Name,
			&column.Position,
			&column.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: scan column: %w", err)
		}
		columns = append(columns, &column)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate columns: %w", err)
	}

	return columns, nil
}

// PATCH /columns/{columnID}
func (r *ColumnRepository) Update(
	ctx context.Context,
	id int,
	name *string,
) (*model.Column, error) {
	var column model.Column
	err := r.db.QueryRow(ctx, `
		UPDATE project_column
		SET name = COALESCE($1, name)
		WHERE id = $2
		RETURNING id, project_uuid, name, position, created_at
	`, name, id).Scan(
		&column.ID,
		&column.ProjectUUID,
		&column.Name,
		&column.Position,
		&column.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrColumnNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: update column: %w", err)
	}
	return &column, nil
}

// DELETE /columns/{columnID}
func (r *ColumnRepository) Delete(ctx context.Context, id int) error {
	result, err := r.db.Exec(ctx, `
		DELETE FROM project_column
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("repository: delete column: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrColumnNotFound
	}
	return nil
}

// GET /columns/{columnID}
func (r *ColumnRepository) FindByID(ctx context.Context, id int) (*model.Column, error) {
	var column model.Column
	err := r.db.QueryRow(ctx, `
		SELECT id, project_uuid, name, position, created_at
		FROM project_column
		WHERE id = $1
	`, id).Scan(
		&column.ID,
		&column.ProjectUUID,
		&column.Name,
		&column.Position,
		&column.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrColumnNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find column by id: %w", err)
	}
	return &column, nil
}
