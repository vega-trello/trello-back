//go:build integration
// +build integration

package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupColumnRepo(t *testing.T) (*ColumnRepository, *pgxpool.Pool) {
	t.Helper()
	pool := setupTestPool(t)
	repo := NewColumnRepository(pool)

	t.Cleanup(func() {
		cleanAllTables(t, pool)
	})

	return repo, pool
}

func TestColumnRepository_Create_Success(t *testing.T) {
	repo, pool := setupColumnRepo(t)
	ctx := context.Background()

	projectUUID := createTestProject(t, pool)

	column, err := repo.Create(ctx, projectUUID, "Backlog", 0)
	require.NoError(t, err)
	require.NotNil(t, column)

	assert.NotZero(t, column.ID)
	assert.Equal(t, projectUUID, column.ProjectUUID)
	assert.Equal(t, "Backlog", column.Name)
	assert.Equal(t, 0, column.Position)
	assert.NotZero(t, column.CreatedAt)
}

func TestColumnRepository_Create_MultipleColumns(t *testing.T) {
	repo, pool := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)

	col1, err := repo.Create(ctx, projectUUID, "To Do", 0)
	require.NoError(t, err)

	col2, err := repo.Create(ctx, projectUUID, "In Progress", 1)
	require.NoError(t, err)

	col3, err := repo.Create(ctx, projectUUID, "Done", 2)
	require.NoError(t, err)

	assert.Equal(t, 0, col1.Position)
	assert.Equal(t, 1, col2.Position)
	assert.Equal(t, 2, col3.Position)
	assert.NotEqual(t, col1.ID, col2.ID)
	assert.NotEqual(t, col2.ID, col3.ID)
}

func TestColumnRepository_Create_DifferentProjects(t *testing.T) {
	repo, pool := setupColumnRepo(t)
	ctx := context.Background()

	projectUUID1 := createTestProject(t, pool)
	projectUUID2 := createTestProject(t, pool)

	col1, err := repo.Create(ctx, projectUUID1, "Project 1 Column", 0)
	require.NoError(t, err)

	col2, err := repo.Create(ctx, projectUUID2, "Project 2 Column", 0)
	require.NoError(t, err)

	assert.Equal(t, projectUUID1, col1.ProjectUUID)
	assert.Equal(t, projectUUID2, col2.ProjectUUID)
}

func TestColumnRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, pool := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)

	_, err := repo.Create(ctx, projectUUID, "Column 1", 0)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID, "Column 2", 1)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID, "Column 3", 2)
	require.NoError(t, err)

	columns, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, columns, 3)

	assert.Equal(t, "Column 1", columns[0].Name)
	assert.Equal(t, "Column 2", columns[1].Name)
	assert.Equal(t, "Column 3", columns[2].Name)
}

func TestColumnRepository_FindByProjectUUID_Empty(t *testing.T) {
	repo, pool := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)

	columns, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	assert.Empty(t, columns)
}

func TestColumnRepository_FindByProjectUUID_SortedByPosition(t *testing.T) {
	repo, pool := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)

	_, err := repo.Create(ctx, projectUUID, "Third", 2)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID, "First", 0)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID, "Second", 1)
	require.NoError(t, err)

	columns, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, columns, 3)

	assert.Equal(t, "First", columns[0].Name)
	assert.Equal(t, "Second", columns[1].Name)
	assert.Equal(t, "Third", columns[2].Name)
}

func TestColumnRepository_FindByProjectUUID_Isolation(t *testing.T) {
	repo, pool := setupColumnRepo(t)
	ctx := context.Background()

	projectUUID1 := createTestProject(t, pool)
	projectUUID2 := createTestProject(t, pool)

	_, err := repo.Create(ctx, projectUUID1, "P1 Col 1", 0)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID1, "P1 Col 2", 1)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID2, "P2 Col 1", 0)
	require.NoError(t, err)

	columns, err := repo.FindByProjectUUID(ctx, projectUUID1)
	require.NoError(t, err)
	require.Len(t, columns, 2)

	for _, col := range columns {
		assert.Equal(t, projectUUID1, col.ProjectUUID)
	}
}

func TestColumnRepository_FindByID_Success(t *testing.T) {
	repo, pool := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)

	created, err := repo.Create(ctx, projectUUID, "Find Me", 0)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)

	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.ProjectUUID, found.ProjectUUID)
	assert.Equal(t, "Find Me", found.Name)
}

func TestColumnRepository_FindByID_NotFound(t *testing.T) {
	repo, _ := setupColumnRepo(t)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, 99999)
	assert.ErrorIs(t, err, ErrColumnNotFound)
}

func TestColumnRepository_Update_Success(t *testing.T) {
	repo, pool := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)

	created, err := repo.Create(ctx, projectUUID, "Old Name", 0)
	require.NoError(t, err)

	newName := "New Name"
	updated, err := repo.Update(ctx, created.ID, &newName)
	require.NoError(t, err)

	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, created.ProjectUUID, updated.ProjectUUID)
	assert.Equal(t, created.Position, updated.Position)
}

func TestColumnRepository_Update_WithNil(t *testing.T) {
	repo, pool := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)

	created, err := repo.Create(ctx, projectUUID, "Keep This Name", 0)
	require.NoError(t, err)

	updated, err := repo.Update(ctx, created.ID, nil)
	require.NoError(t, err)

	assert.Equal(t, "Keep This Name", updated.Name)
	assert.Equal(t, created.ID, updated.ID)
}

func TestColumnRepository_Update_NotFound(t *testing.T) {
	repo, _ := setupColumnRepo(t)
	ctx := context.Background()

	newName := "New Name"
	_, err := repo.Update(ctx, 99999, &newName)
	assert.ErrorIs(t, err, ErrColumnNotFound)
}

func TestColumnRepository_Delete_Success(t *testing.T) {
	repo, pool := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)

	created, err := repo.Create(ctx, projectUUID, "To Delete", 0)
	require.NoError(t, err)

	err = repo.Delete(ctx, created.ID)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, created.ID)
	assert.ErrorIs(t, err, ErrColumnNotFound)
}

func TestColumnRepository_Delete_NotFound(t *testing.T) {
	repo, _ := setupColumnRepo(t)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	assert.ErrorIs(t, err, ErrColumnNotFound)
}

func TestColumnRepository_Delete_MultipleColumns(t *testing.T) {
	repo, pool := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)

	col1, err := repo.Create(ctx, projectUUID, "Col 1", 0)
	require.NoError(t, err)
	col2, err := repo.Create(ctx, projectUUID, "Col 2", 1)
	require.NoError(t, err)
	col3, err := repo.Create(ctx, projectUUID, "Col 3", 2)
	require.NoError(t, err)

	err = repo.Delete(ctx, col2.ID)
	require.NoError(t, err)

	columns, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, columns, 2)

	assert.Equal(t, col1.ID, columns[0].ID)
	assert.Equal(t, col3.ID, columns[1].ID)
}

func TestColumnRepository_FullWorkflow(t *testing.T) {
	repo, pool := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)

	col1, err := repo.Create(ctx, projectUUID, "Backlog", 0)
	require.NoError(t, err)

	col2, err := repo.Create(ctx, projectUUID, "In Progress", 1)
	require.NoError(t, err)

	columns, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, columns, 2)

	newName := "Development"
	updated, err := repo.Update(ctx, col2.ID, &newName)
	require.NoError(t, err)
	assert.Equal(t, "Development", updated.Name)

	found, err := repo.FindByID(ctx, col2.ID)
	require.NoError(t, err)
	assert.Equal(t, "Development", found.Name)

	err = repo.Delete(ctx, col1.ID)
	require.NoError(t, err)

	columns, err = repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, columns, 1)
	assert.Equal(t, col2.ID, columns[0].ID)
}
