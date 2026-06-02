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

func colorPtr(c string) *string { return &c }

func TestColumnRepository_Create_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	col, err := repo.Create(ctx, projectUUID, owner, "Backlog", intPtr(0), nil)
	require.NoError(t, err)
	require.NotNil(t, col)

	assert.Equal(t, "Backlog", col.Name)
	assert.Equal(t, 0, col.Position)
	assert.Equal(t, projectUUID, col.ProjectUUID)
	assert.False(t, col.CreatedAt.IsZero())
	assert.Nil(t, col.Color)
}

func TestColumnRepository_Create_WithColor_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	color := "#FF5733"
	col, err := repo.Create(ctx, projectUUID, owner, "Colored", intPtr(0), colorPtr(color))
	require.NoError(t, err)
	require.NotNil(t, col)

	assert.Equal(t, "Colored", col.Name)
	assert.Equal(t, color, *col.Color)
}

func TestColumnRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	_, _ = repo.Create(ctx, projectUUID, owner, "Col 1", intPtr(0), nil)
	_, _ = repo.Create(ctx, projectUUID, owner, "Col 2", intPtr(1), nil)

	cols, err := repo.FindByProjectUUID(ctx, projectUUID, owner)
	require.NoError(t, err)
	require.Len(t, cols, 2)
	assert.Equal(t, "Col 1", cols[0].Name)
	assert.Equal(t, "Col 2", cols[1].Name)
	assert.Equal(t, 0, cols[0].Position)
	assert.Equal(t, 1, cols[1].Position)
	assert.Nil(t, cols[0].Color)
	assert.Nil(t, cols[1].Color)
}

func TestColumnRepository_FindByID_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	color := "#00FF00"
	created, _ := repo.Create(ctx, projectUUID, owner, "Find Me", intPtr(5), colorPtr(color))

	col, err := repo.FindByID(ctx, created.ID, owner)
	require.NoError(t, err)
	assert.Equal(t, created.ID, col.ID)
	assert.Equal(t, "Find Me", col.Name)
	assert.Equal(t, 5, col.Position)
	assert.Equal(t, color, *col.Color)
}

func TestColumnRepository_FindByID_NotFound(t *testing.T) {
	repo, _, owner := setupColumnRepo(t)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, 99999, owner)
	assert.ErrorIs(t, err, ErrColumnNotFound)
}

func TestColumnRepository_Update_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	col, _ := repo.Create(ctx, projectUUID, owner, "Old Name", intPtr(0), nil)

	newName := "New Name"
	newPos := 10
	updated, err := repo.Update(ctx, col.ID, owner, newName, &newPos, nil)
	require.NoError(t, err)

	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, 10, updated.Position)
	assert.Nil(t, updated.Color)
}

func TestColumnRepository_Update_WithColor_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	col, _ := repo.Create(ctx, projectUUID, owner, "No Color", intPtr(0), nil)
	assert.Nil(t, col.Color)

	newColor := "#123456"
	updated, err := repo.Update(ctx, col.ID, owner, col.Name, nil, colorPtr(newColor))
	require.NoError(t, err)

	assert.Equal(t, newColor, *updated.Color)
}

func TestColumnRepository_Delete_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	col, _ := repo.Create(ctx, projectUUID, owner, "ToDelete", intPtr(0), nil)

	err := repo.Delete(ctx, col.ID, owner)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, col.ID, owner)
	assert.ErrorIs(t, err, ErrColumnNotFound)
}

func TestColumnRepository_Move_Left_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	col1, _ := repo.Create(ctx, projectUUID, owner, "Left", intPtr(0), nil)
	col2, _ := repo.Create(ctx, projectUUID, owner, "Right", intPtr(1), nil)

	// Двигаем правую колонку влево
	updated, err := repo.Move(ctx, col2.ID, owner, "left")
	require.NoError(t, err)
	assert.Equal(t, 0, updated.Position)

	// Проверяем, что левая сдвинулась вправо
	updated1, _ := repo.FindByID(ctx, col1.ID, owner)
	assert.Equal(t, 1, updated1.Position)
}

func TestColumnRepository_Move_Right_Success(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	col1, _ := repo.Create(ctx, projectUUID, owner, "Left", intPtr(0), nil)
	col2, _ := repo.Create(ctx, projectUUID, owner, "Right", intPtr(1), nil)

	// Двигаем левую колонку вправо
	updated, err := repo.Move(ctx, col1.ID, owner, "right")
	require.NoError(t, err)
	assert.Equal(t, 1, updated.Position)

	// Проверяем, что правая сдвинулась влево
	updated2, _ := repo.FindByID(ctx, col2.ID, owner)
	assert.Equal(t, 0, updated2.Position)
}

func TestColumnRepository_Move_Boundary_Left(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	col, _ := repo.Create(ctx, projectUUID, owner, "Only", intPtr(0), nil)

	// Попытка сдвинуть влево, когда колонка уже на позиции 0
	updated, err := repo.Move(ctx, col.ID, owner, "left")
	require.NoError(t, err)
	// Должна остаться на месте, так как соседа слева нет
	assert.Equal(t, 0, updated.Position)
}

func TestColumnRepository_Move_Boundary_Right(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	col, _ := repo.Create(ctx, projectUUID, owner, "Only", intPtr(5), nil)

	// Попытка сдвинуть вправо, когда соседа справа нет
	updated, err := repo.Move(ctx, col.ID, owner, "right")
	require.NoError(t, err)
	assert.Equal(t, 5, updated.Position)
}

func TestColumnRepository_Move_WithColor(t *testing.T) {
	repo, pool, owner := setupColumnRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	color := "#ABCDEF"
	col1, _ := repo.Create(ctx, projectUUID, owner, "Left", intPtr(0), colorPtr(color))
	col2, _ := repo.Create(ctx, projectUUID, owner, "Right", intPtr(1), nil)

	// Двигаем правую колонку влево
	updated, err := repo.Move(ctx, col2.ID, owner, "left")
	require.NoError(t, err)
	assert.Equal(t, 0, updated.Position)
	assert.Nil(t, updated.Color) // У col2 не было цвета

	// Проверяем, что у сдвинутой колонки цвет сохранился
	updated1, _ := repo.FindByID(ctx, col1.ID, owner)
	assert.Equal(t, color, *updated1.Color)
}
