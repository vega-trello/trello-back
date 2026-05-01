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

func setupTagRepo(t *testing.T) (*TagRepository, *pgxpool.Pool, uuid.UUID) {
	t.Helper()
	pool := setupTestPool(t)
	owner := createTestUser(t, pool, "tag_owner", "pass123")
	repo := NewTagRepository(pool)
	return repo, pool, owner
}

func TestTagRepository_Create_Success(t *testing.T) {
	repo, pool, owner := setupTagRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	tag, err := repo.Create(ctx, projectUUID, owner, "Backend", 0xFF5733)
	require.NoError(t, err)
	require.NotNil(t, tag)

	assert.Equal(t, "Backend", tag.Name)
	assert.Equal(t, 0xFF5733, tag.Color)
	assert.Equal(t, projectUUID, tag.ProjectUUID)
	assert.Greater(t, tag.ID, 0)
}

func TestTagRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, pool, owner := setupTagRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	_, _ = repo.Create(ctx, projectUUID, owner, "API", 0x0000FF)
	_, _ = repo.Create(ctx, projectUUID, owner, "UI", 0x00FF00)

	tags, err := repo.FindByProjectUUID(ctx, projectUUID, owner)
	require.NoError(t, err)
	require.Len(t, tags, 2)
}

func TestTagRepository_Update_Success(t *testing.T) {
	repo, pool, owner := setupTagRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	tag, _ := repo.Create(ctx, projectUUID, owner, "OldName", 0x111111)

	newName := "NewName"
	newColor := 0x999999
	updated, err := repo.Update(ctx, tag.ID, owner, &newName, &newColor)
	require.NoError(t, err)

	assert.Equal(t, "NewName", updated.Name)
	assert.Equal(t, 0x999999, updated.Color)
}

func TestTagRepository_Delete_Success(t *testing.T) {
	repo, pool, owner := setupTagRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	tag, _ := repo.Create(ctx, projectUUID, owner, "ToDelete", 0xFF0000)

	err := repo.Delete(ctx, tag.ID, owner)
	assert.NoError(t, err)

	_, err = repo.FindByProjectUUID(ctx, projectUUID, owner)
	require.NoError(t, err)
}

func TestTagRepository_AddToTask_Success(t *testing.T) {
	repo, pool, owner := setupTagRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	columnID := createTestColumn(t, pool, projectUUID, 0)

	var taskID int
	pool.QueryRow(ctx, `INSERT INTO tasks (column_id, creator_uuid, title, created_at, updated_at) VALUES ($1, $2, 'Task', NOW(), NOW()) RETURNING id`, columnID, owner).Scan(&taskID)

	tag, _ := repo.Create(ctx, projectUUID, owner, "AddedTag", 0x555555)

	err := repo.AddToTask(ctx, projectUUID, owner, taskID, tag.ID)
	assert.NoError(t, err)

	tags, _ := repo.FindByTask(ctx, taskID, owner)
	require.Len(t, tags, 1)
	assert.Equal(t, "AddedTag", tags[0].Name)
}

func TestTagRepository_RemoveFromTask_Success(t *testing.T) {
	repo, pool, owner := setupTagRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	columnID := createTestColumn(t, pool, projectUUID, 0)

	var taskID int
	pool.QueryRow(ctx, `INSERT INTO tasks (column_id, creator_uuid, title, created_at, updated_at) VALUES ($1, $2, 'Task', NOW(), NOW()) RETURNING id`, columnID, owner).Scan(&taskID)

	tag, _ := repo.Create(ctx, projectUUID, owner, "ToRemove", 0x777777)
	_ = repo.AddToTask(ctx, projectUUID, owner, taskID, tag.ID)

	err := repo.RemoveFromTask(ctx, projectUUID, owner, taskID, tag.ID)
	assert.NoError(t, err)

	tags, _ := repo.FindByTask(ctx, taskID, owner)
	assert.Empty(t, tags)
}
