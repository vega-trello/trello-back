//go:build integration
// +build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dto "github.com/vega-trello/trello-back/internal/dto/project"
)

func setupProjectRepo(t *testing.T) (*ProjectRepository, *pgxpool.Pool, uuid.UUID) {
	t.Helper()
	pool := setupTestPool(t)
	repo := NewProjectRepository(pool)
	owner := createTestUser(t, pool, "project_owner", "pass123")
	return repo, pool, owner
}

func TestProjectRepository_Create_Success(t *testing.T) {
	repo, _, owner := setupProjectRepo(t)
	ctx := context.Background()

	req := dto.CreateProjectRequest{
		Title:       "Test Project",
		Description: "Description",
	}

	project, err := repo.Create(ctx, owner, req)
	require.NoError(t, err)
	require.NotNil(t, project)

	assert.NotEqual(t, uuid.Nil, project.UUID)
	assert.Equal(t, "Test Project", project.Title)
	assert.Equal(t, "Description", project.Description)
	assert.WithinDuration(t, time.Now(), project.CreatedAt, time.Second)
}

func TestProjectRepository_FindByID_Success(t *testing.T) {
	repo, _, owner := setupProjectRepo(t)
	ctx := context.Background()

	req := dto.CreateProjectRequest{Title: "Find Me", Description: "Desc"}
	created, err := repo.Create(ctx, owner, req)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, created.UUID)
	require.NoError(t, err)
	assert.Equal(t, created.UUID, found.UUID)
	assert.Equal(t, created.Title, found.Title)
}

func TestProjectRepository_Update_Success(t *testing.T) {
	repo, _, owner := setupProjectRepo(t)
	ctx := context.Background()

	req := dto.CreateProjectRequest{Title: "Old Title", Description: "Old Desc"}
	created, err := repo.Create(ctx, owner, req)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	newTitle := "New Title"
	newDesc := "New Desc"
	updated, err := repo.Update(ctx, created.UUID, &newTitle, &newDesc)
	require.NoError(t, err)

	assert.Equal(t, "New Title", updated.Title)
	assert.Equal(t, "New Desc", updated.Description)
	assert.True(t, updated.UpdatedAt.After(updated.CreatedAt))
}

func TestProjectRepository_Delete_Success(t *testing.T) {
	repo, _, owner := setupProjectRepo(t)
	ctx := context.Background()

	req := dto.CreateProjectRequest{Title: "To Delete", Description: "Desc"}
	created, err := repo.Create(ctx, owner, req)
	require.NoError(t, err)

	err = repo.Delete(ctx, created.UUID)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, created.UUID)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestProjectRepository_UserHasAccess_Success(t *testing.T) {
	repo, _, owner := setupProjectRepo(t)
	ctx := context.Background()

	req := dto.CreateProjectRequest{Title: "Access Test", Description: "Desc"}
	project, err := repo.Create(ctx, owner, req)
	require.NoError(t, err)

	hasAccess, err := repo.UserHasAccess(ctx, owner, project.UUID)
	require.NoError(t, err)
	assert.True(t, hasAccess)
}
