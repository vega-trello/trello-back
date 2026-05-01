package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	dto "github.com/vega-trello/trello-back/internal/dto/project"
	"github.com/vega-trello/trello-back/internal/model"
)

type ProjectRepositoryInterface interface {
	Create(ctx context.Context, creatorUUID uuid.UUID, req dto.CreateProjectRequest) (*model.Project, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
	Update(ctx context.Context, id uuid.UUID, title, description *string) (*model.Project, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UserHasAccess(ctx context.Context, userUUID, projectUUID uuid.UUID) (bool, error)
}

type ProjectRepository struct {
	db *pgxpool.Pool
}

func NewProjectRepository(db *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create создаёт проект и добавляет создателя как владельца (роль ID=1)
func (r *ProjectRepository) Create(
	ctx context.Context,
	creatorUUID uuid.UUID,
	req dto.CreateProjectRequest,
) (*model.Project, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	projectUUID := uuid.New()
	now := time.Now()

	_, err = tx.Exec(ctx, `
		INSERT INTO project (uuid, title, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
	`, projectUUID, req.Title, req.Description, now)
	if err != nil {
		return nil, fmt.Errorf("repository: insert project: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO project_member (project_uuid, user_uuid, role_id, joined_at)
		VALUES ($1, $2, 1, $3)
	`, projectUUID, creatorUUID, now)
	if err != nil {
		return nil, fmt.Errorf("repository: insert project member: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return &model.Project{
		UUID:        projectUUID,
		Title:       req.Title,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// FindByID находит проект по UUID
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

// Update обновляет название и/или описание проекта (частичное обновление)
func (r *ProjectRepository) Update(
	ctx context.Context,
	id uuid.UUID,
	title, description *string,
) (*model.Project, error) {
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

// Delete удаляет проект (каскадно удаляет связанные данные через ON DELETE CASCADE)
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

func (r *ProjectRepository) UserHasAccess(
	ctx context.Context,
	userUUID, projectUUID uuid.UUID,
) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM project_member
			WHERE project_uuid = $1 AND user_uuid = $2
		)
	`, projectUUID, userUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("repository: check user access: %w", err)
	}
	return exists, nil
}
