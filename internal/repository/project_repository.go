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

var ErrProjectNotFound = errors.New("project not found")

type ProjectRepositoryInterface interface {
	Create(ctx context.Context, title, description string) (*model.Project, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
	Update(ctx context.Context, id uuid.UUID, title, description *string) (*model.Project, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ProjectRepository struct {
	db *pgxpool.Pool
}

func NewProjectRepository(db *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(ctx context.Context, title, description string) (*model.Project, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var project model.Project
	err = tx.QueryRow(ctx, `
		INSERT INTO project (uuid, title, description, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, NOW(), NOW())
		RETURNING uuid, title, description, created_at, updated_at
	`, title, description).Scan(
		&project.UUID,
		&project.Title,
		&project.Description,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: create project: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return &project, nil
}

func (r *ProjectRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	var project model.Project
	err := r.db.QueryRow(ctx, `
		SELECT uuid, title, description, created_at, updated_at
		FROM project
		WHERE uuid = $1
	`, id).Scan(
		&project.UUID,
		&project.Title,
		&project.Description,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find project by id: %w", err)
	}
	return &project, nil
}

func (r *ProjectRepository) Update(ctx context.Context, id uuid.UUID, title, description *string) (*model.Project, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var project model.Project
	err = tx.QueryRow(ctx, `
		UPDATE project
		SET
			title = COALESCE($1, title),
			description = COALESCE($2, description),
			updated_at = NOW()
		WHERE uuid = $3
		RETURNING uuid, title, description, created_at, updated_at
	`, title, description, id).Scan(
		&project.UUID,
		&project.Title,
		&project.Description,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: update project: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &project, nil
}

func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `
		DELETE FROM project
		WHERE uuid = $1
	`, id)
	if err != nil {
		return fmt.Errorf("repository: delete project: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrProjectNotFound
	}
	return nil
}
