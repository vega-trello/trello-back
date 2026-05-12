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

var (
	ErrStatusNotFound = errors.New("project status not found")
)

type StatusRepository struct {
	db *pgxpool.Pool
}

func NewStatusRepository(db *pgxpool.Pool) *StatusRepository {
	return &StatusRepository{db: db}
}

func (r *StatusRepository) Create(
	ctx context.Context,
	projectUUID uuid.UUID,
	name string,
	callerUUID uuid.UUID,
) (*model.ProjectStatus, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.ensureUserAccessTx(ctx, tx, projectUUID, callerUUID); err != nil {
		return nil, err
	}

	var status model.ProjectStatus
	err = tx.QueryRow(ctx, `
		INSERT INTO project_status (project_uuid, name, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id, project_uuid, name, created_at
	`, projectUUID, name).Scan(&status.ID, &status.ProjectUUID, &status.Name, &status.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("repository: create status: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &status, nil
}

func (r *StatusRepository) FindByProject(
	ctx context.Context,
	projectUUID uuid.UUID,
	callerUUID uuid.UUID,
) ([]*model.ProjectStatus, error) {
	if err := r.ensureUserAccess(ctx, projectUUID, callerUUID); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, project_uuid, name, created_at
		FROM project_status
		WHERE project_uuid = $1
		ORDER BY created_at ASC
	`, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("repository: find statuses by project: %w", err)
	}
	defer rows.Close()

	var statuses []*model.ProjectStatus
	for rows.Next() {
		var s model.ProjectStatus
		if err := rows.Scan(&s.ID, &s.ProjectUUID, &s.Name, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("repository: scan status: %w", err)
		}
		statuses = append(statuses, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate statuses: %w", err)
	}
	return statuses, nil
}

func (r *StatusRepository) FindByID(
	ctx context.Context,
	projectUUID uuid.UUID,
	statusID int,
	callerUUID uuid.UUID,
) (*model.ProjectStatus, error) {
	if err := r.ensureUserAccess(ctx, projectUUID, callerUUID); err != nil {
		return nil, err
	}
	var status model.ProjectStatus
	err := r.db.QueryRow(ctx, `
		SELECT id, project_uuid, name, created_at
		FROM project_status
		WHERE id = $1 AND project_uuid = $2
	`, statusID, projectUUID).Scan(&status.ID, &status.ProjectUUID, &status.Name, &status.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStatusNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find status by id: %w", err)
	}
	return &status, nil
}

func (r *StatusRepository) Update(
	ctx context.Context,
	projectUUID uuid.UUID,
	statusID int,
	newName string,
	callerUUID uuid.UUID,
) (*model.ProjectStatus, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.ensureUserAccessTx(ctx, tx, projectUUID, callerUUID); err != nil {
		return nil, err
	}

	var status model.ProjectStatus
	err = tx.QueryRow(ctx, `
		UPDATE project_status
		SET name = $1
		WHERE id = $2 AND project_uuid = $3
		RETURNING id, project_uuid, name, created_at
	`, newName, statusID, projectUUID).Scan(&status.ID, &status.ProjectUUID, &status.Name, &status.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStatusNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: update status: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}
	return &status, nil
}

func (r *StatusRepository) Delete(
	ctx context.Context,
	projectUUID uuid.UUID,
	statusID int,
	callerUUID uuid.UUID,
) error {
	if err := r.ensureUserAccess(ctx, projectUUID, callerUUID); err != nil {
		return err
	}
	result, err := r.db.Exec(ctx, `
		DELETE FROM project_status 
		WHERE id = $1 AND project_uuid = $2
	`, statusID, projectUUID)
	if err != nil {
		return fmt.Errorf("repository: delete status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrStatusNotFound
	}
	return nil
}

func (r *StatusRepository) ensureUserAccess(ctx context.Context, projectUUID, userUUID uuid.UUID) error {
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

func (r *StatusRepository) ensureUserAccessTx(ctx context.Context, tx pgx.Tx, projectUUID, userUUID uuid.UUID) error {
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
