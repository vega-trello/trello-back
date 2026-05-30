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

var (
	ErrProjectAlreadyExists = errors.New("project with this UUID already exists")
)

type ProjectRepository struct {
	db *pgxpool.Pool
}

func NewProjectRepository(db *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create создаёт проект и добавляет создателя как владельца (роль ID=1)
func (r *ProjectRepository) Create(
	ctx context.Context,
	userUUID uuid.UUID,
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
		if isUniqueViolation(err, "project_uuid_key") {
			return nil, ErrProjectAlreadyExists
		}
		return nil, fmt.Errorf("repository: insert project: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO project_member (project_uuid, user_uuid, role_id, joined_at)
		VALUES ($1, $2, 1, $3)
	`, projectUUID, userUUID, now)
	if err != nil {
		return nil, fmt.Errorf("repository: insert project member: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
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

// FindByID находит проект по UUID (без проверки прав это делает сервис)
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

// FindByUser возвращает все проекты, в которых пользователь является участником
func (r *ProjectRepository) FindByUser(
	ctx context.Context,
	userUUID uuid.UUID,
) ([]*model.Project, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.uuid, p.title, p.description, p.created_at, p.updated_at
		FROM project p
		JOIN project_member pm ON p.uuid = pm.project_uuid
		WHERE pm.user_uuid = $1
		ORDER BY p.created_at DESC
	`, userUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find projects by user: %w", err)
	}
	defer rows.Close()

	var projects []*model.Project
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.UUID, &p.Title, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan project: %w", err)
		}
		projects = append(projects, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate projects: %w", err)
	}
	return projects, nil
}

// Update обновляет название и/или описание проекта (частичное обновление)
// *string позволяет передать nil = "не менять поле"
// Проверка прав доступа выполняется внутри метода
func (r *ProjectRepository) Update(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	title *string,
	description *string,
) (*model.Project, error) {
	isMember, err := r.IsMember(ctx, projectUUID, userUUID)
	if err != nil || !isMember {
		return nil, ErrAccessDenied
	}

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
	`, title, description, projectUUID).Scan(
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

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &project, nil
}

// Delete удаляет проект (каскадно удаляет связанные данные через ON DELETE CASCADE)
// Проверка прав (только владелец) выполняется внутри метода
func (r *ProjectRepository) Delete(ctx context.Context, projectUUID uuid.UUID) error {
	result, err := r.db.Exec(ctx, `
		DELETE FROM project
		WHERE uuid = $1
	`, projectUUID)
	if err != nil {
		return fmt.Errorf("repository: delete project: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// IsMember проверяет, является ли пользователь участником проекта
func (r *ProjectRepository) IsMember(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM project_member
			WHERE project_uuid = $1 AND user_uuid = $2
		)
	`, projectUUID, userUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("repository: check member: %w", err)
	}
	return exists, nil
}

// IsOwner проверяет, является ли пользователь владельцем проекта (роль = 1)
func (r *ProjectRepository) IsOwner(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM project_member
			WHERE project_uuid = $1 AND user_uuid = $2 AND role_id = 1
		)
	`, projectUUID, userUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("repository: check owner: %w", err)
	}
	return exists, nil
}

// RemoveMember удаляет пользователя из проекта (выход из проекта)
func (r *ProjectRepository) RemoveMember(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) error {
	result, err := r.db.Exec(ctx, `
		DELETE FROM project_member
		WHERE project_uuid = $1 AND user_uuid = $2
	`, projectUUID, userUUID)
	if err != nil {
		return fmt.Errorf("repository: remove member: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrAccessDenied // пользователь не был участником
	}
	return nil
}
