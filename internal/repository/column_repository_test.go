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

func setupColumnRepo(t *testing.T) (*ColumnRepository, *pgxpool.Pool, uuid.UUID) {
	t.Helper()
	pool := setupTestPool(t)
	owner := createTestUser(t, pool, "column_owner", "pass123")
	repo := NewColumnRepository(pool)
	return repo, pool, owner
}

func TestColumnRepository_Create_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	col, err := repo.Create(ctx, projectUUID, owner, "Backlog", intPtr(0))
	require.NoError(t, err)
	require.NotNil(t, col)

	assert.Equal(t, "Backlog", col.Name)
	assert.Equal(t, 0, col.Position)
	assert.Equal(t, projectUUID, col.ProjectUUID)
	assert.False(t, col.CreatedAt.IsZero())
}

func TestColumnRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	_, _ = repo.Create(ctx, projectUUID, owner, "Col 1", intPtr(0))
	_, _ = repo.Create(ctx, projectUUID, owner, "Col 2", intPtr(1))

	cols, err := repo.FindByProjectUUID(ctx, projectUUID, owner)
	require.NoError(t, err)
	require.Len(t, cols, 2)
	assert.Equal(t, "Col 1", cols[0].Name)
	assert.Equal(t, "Col 2", cols[1].Name)
}

func TestColumnRepository_FindByID_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	created, _ := repo.Create(ctx, projectUUID, owner, "Find Me", intPtr(5))

	col, err := repo.FindByID(ctx, created.ID, owner)
	require.NoError(t, err)
	assert.Equal(t, created.ID, col.ID)
	assert.Equal(t, "Find Me", col.Name)
	assert.Equal(t, 5, col.Position)
}

func TestColumnRepository_Update_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	col, _ := repo.Create(ctx, projectUUID, owner, "Old Name", intPtr(0))

	newName := "New Name"
	newPos := 10
	updated, err := repo.Update(ctx, col.ID, owner, newName, &newPos)
	require.NoError(t, err)

	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, 10, updated.Position)
}

func TestColumnRepository_Delete_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	col, _ := repo.Create(ctx, projectUUID, owner, "ToDelete", intPtr(0))

	err := repo.Delete(ctx, col.ID, owner)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, col.ID, owner)
	assert.ErrorIs(t, err, ErrColumnNotFound)
}

func TestColumnRepository_Move_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	col, _ := repo.Create(ctx, projectUUID, owner, "ToMove", intPtr(0))

	updated, err := repo.Move(ctx, col.ID, owner, 5)
	require.NoError(t, err)
	assert.Equal(t, 5, updated.Position)
}
