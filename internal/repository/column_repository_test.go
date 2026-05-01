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
	repo := NewColumnRepository(pool)
	owner := createTestUser(t, pool, "column_owner", "pass123")
	return repo, pool, owner
}

func TestColumnRepository_Create_Success(t *testing.T) {
	repo, _, owner := setupColumnRepo(t)
	ctx := context.Background()
	pool := setupTestPool(t) // нужен только для создания проекта
	projectUUID := createTestProject(t, pool, owner)

	col, err := repo.Create(ctx, projectUUID, owner, "Backlog", intPtr(0))
	require.NoError(t, err)
	require.NotNil(t, col)

	assert.Equal(t, "Backlog", col.Name)
	assert.Equal(t, 0, col.Position)
	assert.Equal(t, projectUUID, col.ProjectUUID)
}

func TestColumnRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, _, owner := setupColumnRepo(t)
	ctx := context.Background()
	pool := setupTestPool(t)
	projectUUID := createTestProject(t, pool, owner)

	_, _ = repo.Create(ctx, projectUUID, owner, "Col 1", intPtr(0))
	_, _ = repo.Create(ctx, projectUUID, owner, "Col 2", intPtr(1))

	cols, err := repo.FindByProjectUUID(ctx, projectUUID, owner)
	require.NoError(t, err)
	require.Len(t, cols, 2)
}

func TestColumnRepository_Update_Success(t *testing.T) {
	repo, _, owner := setupColumnRepo(t)
	ctx := context.Background()
	pool := setupTestPool(t)
	projectUUID := createTestProject(t, pool, owner)

	col, _ := repo.Create(ctx, projectUUID, owner, "Old Name", intPtr(0))

	newName := "New Name"
	newPos := 5
	updated, err := repo.Update(ctx, col.ID, owner, &newName, &newPos)
	require.NoError(t, err)

	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, 5, updated.Position)
}

func TestColumnRepository_Delete_Success(t *testing.T) {
	repo, _, owner := setupColumnRepo(t)
	ctx := context.Background()
	pool := setupTestPool(t)
	projectUUID := createTestProject(t, pool, owner)

	col, _ := repo.Create(ctx, projectUUID, owner, "To Delete", intPtr(0))

	err := repo.Delete(ctx, col.ID, owner)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, col.ID, owner)
	assert.ErrorIs(t, err, ErrColumnNotFound)
}

func TestColumnRepository_Move_Success(t *testing.T) {
	repo, _, owner := setupColumnRepo(t)
	ctx := context.Background()
	pool := setupTestPool(t)
	projectUUID := createTestProject(t, pool, owner)

	col1, _ := repo.Create(ctx, projectUUID, owner, "First", intPtr(0))
	col2, _ := repo.Create(ctx, projectUUID, owner, "Second", intPtr(1))
	col3, _ := repo.Create(ctx, projectUUID, owner, "Third", intPtr(2))

	updated, err := repo.Move(ctx, col1.ID, owner, 2)
	require.NoError(t, err)
	assert.Equal(t, 2, updated.Position)

	cols, _ := repo.FindByProjectUUID(ctx, projectUUID, owner)
	require.Len(t, cols, 3)
	assert.Equal(t, col2.ID, cols[0].ID)
	assert.Equal(t, col3.ID, cols[1].ID)
	assert.Equal(t, col1.ID, cols[2].ID)
}
