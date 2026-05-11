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
)

func setupTaskRepo(t *testing.T) (*TaskRepository, *pgxpool.Pool, uuid.UUID) {
	t.Helper()
	pool := setupTestPool(t)
	owner := createTestUser(t, pool, "task_owner", "pass123")
	repo := NewTaskRepository(pool)
	return repo, pool, owner
}

func TestTaskRepository_Create_Success(t *testing.T) {
	repo, pool, owner := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	columnID := createTestColumn(t, pool, projectUUID, 0)

	task, err := repo.Create(ctx, projectUUID, columnID, owner, "New Task", "Task description", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, task)

	assert.Equal(t, columnID, task.ColumnID)
	assert.Equal(t, "New Task", task.Title)
	assert.Equal(t, owner, task.CreatorUUID)
	assert.False(t, task.CreatedAt.IsZero())
	assert.False(t, task.UpdatedAt.IsZero())
}

func TestTaskRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, pool, owner := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	columnID := createTestColumn(t, pool, projectUUID, 0)

	_, _ = repo.Create(ctx, projectUUID, columnID, owner, "Task 1", "Desc 1", nil, nil)
	time.Sleep(10 * time.Millisecond)
	_, _ = repo.Create(ctx, projectUUID, columnID, owner, "Task 2", "Desc 2", nil, nil)

	tasks, err := repo.FindByProjectUUID(ctx, projectUUID, owner, nil)
	require.NoError(t, err)
	require.Len(t, tasks, 2)
}

func TestTaskRepository_FindByID_Success(t *testing.T) {
	repo, pool, owner := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	columnID := createTestColumn(t, pool, projectUUID, 0)

	created, _ := repo.Create(ctx, projectUUID, columnID, owner, "Find Me", "Desc", nil, nil)

	task, err := repo.FindByID(ctx, projectUUID, created.ID, owner)
	require.NoError(t, err)
	assert.Equal(t, created.ID, task.ID)
	assert.Equal(t, "Find Me", task.Title)
}

func TestTaskRepository_Update_Success(t *testing.T) {
	repo, pool, owner := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	columnID := createTestColumn(t, pool, projectUUID, 0)

	task, _ := repo.Create(ctx, projectUUID, columnID, owner, "Old Title", "Old Desc", nil, nil)
	oldUpdatedAt := task.UpdatedAt

	newTitle := "New Title"
	newDesc := "Updated description"
	updated, err := repo.Update(ctx, projectUUID, task.ID, owner, &newTitle, &newDesc, nil, nil, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "New Title", updated.Title)
	assert.Equal(t, "Updated description", updated.Description)
	assert.True(t, updated.UpdatedAt.After(oldUpdatedAt))
}

func TestTaskRepository_Delete_Success(t *testing.T) {
	repo, pool, owner := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	columnID := createTestColumn(t, pool, projectUUID, 0)

	task, _ := repo.Create(ctx, projectUUID, columnID, owner, "To Delete", "Desc", nil, nil)

	err := repo.Delete(ctx, projectUUID, task.ID, owner)
	require.NoError(t, err)

	_, err = repo.FindByID(ctx, projectUUID, task.ID, owner)
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestTaskRepository_Move_Success(t *testing.T) {
	repo, pool, owner := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	col1 := createTestColumn(t, pool, projectUUID, 0)
	col2 := createTestColumn(t, pool, projectUUID, 1)

	task, _ := repo.Create(ctx, projectUUID, col1, owner, "Move Me", "Desc", nil, nil)

	err := repo.Move(ctx, projectUUID, task.ID, col2, owner)
	require.NoError(t, err)

	updated, _ := repo.FindByID(ctx, projectUUID, task.ID, owner)
	assert.Equal(t, col2, updated.ColumnID)
}
