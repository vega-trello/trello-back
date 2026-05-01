//go:build integration
// +build integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStatusRepo(t *testing.T) (*StatusRepository, *pgxpool.Pool, uuid.UUID) {
	t.Helper()
	pool := setupTestPool(t)
	owner := createTestUser(t, pool, "status_owner", "pass123")
	repo := NewStatusRepository(pool)
	return repo, pool, owner
}

func TestStatusRepository_Create_Success(t *testing.T) {
	repo, _, owner := setupStatusRepo(t)
	ctx := context.Background()
	pool := setupTestPool(t)
	projectUUID := createTestProject(t, pool, owner)

	status, err := repo.Create(ctx, projectUUID, "In Progress", owner)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, "In Progress", status.Name)
	assert.Equal(t, projectUUID, status.ProjectUUID)
}

func TestStatusRepository_FindByProject_Success(t *testing.T) {
	repo, _, owner := setupStatusRepo(t)
	ctx := context.Background()
	pool := setupTestPool(t)
	projectUUID := createTestProject(t, pool, owner)

	_, _ = repo.Create(ctx, projectUUID, "To Do", owner)
	_, _ = repo.Create(ctx, projectUUID, "Done", owner)

	statuses, err := repo.FindByProject(ctx, projectUUID, owner)
	require.NoError(t, err)
	require.Len(t, statuses, 2)
}

func TestStatusRepository_Update_Success(t *testing.T) {
	repo, _, owner := setupStatusRepo(t)
	ctx := context.Background()
	pool := setupTestPool(t)
	projectUUID := createTestProject(t, pool, owner)

	created, _ := repo.Create(ctx, projectUUID, "Old Name", owner)
	updated, err := repo.Update(ctx, projectUUID, created.ID, "New Name", owner)
	require.NoError(t, err)

	assert.Equal(t, "New Name", updated.Name)
}

func TestStatusRepository_Delete_Success(t *testing.T) {
	repo, _, owner := setupStatusRepo(t)
	ctx := context.Background()
	pool := setupTestPool(t)
	projectUUID := createTestProject(t, pool, owner)

	created, _ := repo.Create(ctx, projectUUID, "ToDelete", owner)
	err := repo.Delete(ctx, projectUUID, created.ID, owner)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, projectUUID, created.ID, owner)
	assert.ErrorIs(t, err, ErrStatusNotFound)
}

func TestStatusRepository_AccessDenied(t *testing.T) {
	repo, _, owner := setupStatusRepo(t)
	ctx := context.Background()
	pool := setupTestPool(t)
	projectUUID := createTestProject(t, pool, owner)

	outsider := createTestUser(t, pool, "outsider", "pass123")

	_, err := repo.Create(ctx, projectUUID, "Secret", outsider)
	assert.ErrorIs(t, err, ErrAccessDenied)

	status, _ := repo.Create(ctx, projectUUID, "TestStatus", owner)
	_, err = repo.FindByID(ctx, projectUUID, status.ID, outsider)
	assert.ErrorIs(t, err, ErrAccessDenied)
}
