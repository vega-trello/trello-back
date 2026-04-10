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

func setupTagRepo(t *testing.T) (*TagRepository, *pgxpool.Pool) {
	t.Helper()
	pool := setupTestPool(t)
	repo := NewTagRepository(pool)

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "TRUNCATE task_tag, tag RESTART IDENTITY CASCADE")
		cleanAllTables(t, pool)
	})

	return repo, pool
}

func TestTagRepository_Create_Success(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	name := "Backend"
	color := 0xFF5733

	tag, err := repo.Create(ctx, projectUUID, name, color)
	require.NoError(t, err)
	require.NotNil(t, tag)

	assert.Equal(t, name, tag.Name)
	assert.Equal(t, color, tag.Color)
	assert.Equal(t, projectUUID, tag.ProjectUUID)
	assert.Greater(t, tag.ID, 0)
	assert.False(t, tag.CreatedAt.IsZero())
}

func TestTagRepository_Create_DuplicateName(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	name := "Bug"
	color := 0xFF0000

	_, err := repo.Create(ctx, projectUUID, name, color)
	require.NoError(t, err)

	_, err = repo.Create(ctx, projectUUID, name, 0x00FF00)
	require.Error(t, err)
}

func TestTagRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	_, _ = repo.Create(ctx, projectUUID, "API", 0x0000FF)
	_, _ = repo.Create(ctx, projectUUID, "UI", 0x00FF00)
	_, _ = repo.Create(ctx, projectUUID, "Bug", 0xFF0000)

	tags, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, tags, 3)

	assert.Equal(t, "API", tags[0].Name)
	assert.Equal(t, "Bug", tags[1].Name)
	assert.Equal(t, "UI", tags[2].Name)
}

func TestTagRepository_FindByProjectUUID_Empty(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	tags, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestTagRepository_Update_Success(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	tag, _ := repo.Create(ctx, projectUUID, "OldName", 0x111111)

	newName := "NewName"
	newColor := 0x999999
	updates := TagUpdates{
		Name:  &newName,
		Color: &newColor,
	}

	updated, err := repo.Update(ctx, projectUUID, tag.ID, updates)
	require.NoError(t, err)

	assert.Equal(t, "NewName", updated.Name)
	assert.Equal(t, 0x999999, updated.Color)
	assert.Equal(t, tag.CreatedAt, updated.CreatedAt)
}

func TestTagRepository_Update_Partial(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	tag, _ := repo.Create(ctx, projectUUID, "KeepName", 0x222222)

	newColor := 0x333333
	updates := TagUpdates{Color: &newColor}

	updated, err := repo.Update(ctx, projectUUID, tag.ID, updates)
	require.NoError(t, err)

	assert.Equal(t, "KeepName", updated.Name) // Имя не изменилось
	assert.Equal(t, 0x333333, updated.Color)
}

func TestTagRepository_Update_NotFound(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	name := "test"
	updates := TagUpdates{Name: &name}

	_, err := repo.Update(ctx, projectUUID, 99999, updates)
	require.Error(t, err)
}

func TestTagRepository_Update_WrongProject(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()

	projectUUID1, _ := createTestProjectAndColumn(t, pool)
	projectUUID2, _ := createTestProjectAndColumn(t, pool)

	tag, _ := repo.Create(ctx, projectUUID1, "Project1Tag", 0xAAAAAA)

	name := "Changed"
	updates := TagUpdates{Name: &name}

	// Пытаемся обновить тег из проекта 1 через проект 2
	_, err := repo.Update(ctx, projectUUID2, tag.ID, updates)
	require.Error(t, err)
}

func TestTagRepository_Update_DuplicateName(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	// Создаём два тега с разными именами
	_, _ = repo.Create(ctx, projectUUID, "TagA", 0x111111)
	tag2, _ := repo.Create(ctx, projectUUID, "TagB", 0x222222)

	// Пытаемся переименовать tag2 ("TagB") в "TagA", который уже занят
	name := "TagA"
	updates := TagUpdates{Name: &name}

	// Это должно вызвать ошибку UNIQUE violation
	_, err := repo.Update(ctx, projectUUID, tag2.ID, updates)
	require.Error(t, err)
}

func TestTagRepository_Delete_Success(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	tag, _ := repo.Create(ctx, projectUUID, "ToDelete", 0xFF0000)

	err := repo.Delete(ctx, projectUUID, tag.ID)
	assert.NoError(t, err)

	// Проверяем, что тег действительно удалён
	_, err = repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	// Список должен быть пуст
	tags, _ := repo.FindByProjectUUID(ctx, projectUUID)
	assert.Empty(t, tags)
}

func TestTagRepository_Delete_NotFound(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	err := repo.Delete(ctx, projectUUID, 99999)
	assert.Error(t, err)
}

func TestTagRepository_Delete_WrongProject(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()

	projectUUID1, _ := createTestProjectAndColumn(t, pool)
	projectUUID2, _ := createTestProjectAndColumn(t, pool)

	tag, _ := repo.Create(ctx, projectUUID1, "WrongProjTag", 0xCCCCCC)

	// Пытаемся удалить через проект 2
	err := repo.Delete(ctx, projectUUID2, tag.ID)
	assert.Error(t, err)
}

func TestTagRepository_FindByTask_Success(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	tag1, _ := repo.Create(ctx, projectUUID, "Frontend", 0x0000FF)
	tag2, _ := repo.Create(ctx, projectUUID, "Critical", 0xFF0000)

	require.NoError(t, repo.AddToTask(ctx, projectUUID, AssignTagInput{TaskID: taskID, TagID: tag1.ID}))
	require.NoError(t, repo.AddToTask(ctx, projectUUID, AssignTagInput{TaskID: taskID, TagID: tag2.ID}))

	tags, err := repo.FindByTask(ctx, projectUUID, taskID)
	require.NoError(t, err)
	require.Len(t, tags, 2)

	// Проверяем сортировку по имени
	assert.Equal(t, "Critical", tags[0].Name)
	assert.Equal(t, "Frontend", tags[1].Name)
}

func TestTagRepository_FindByTask_Empty(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	tags, err := repo.FindByTask(ctx, projectUUID, taskID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestTagRepository_FindByTask_WrongProject(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()

	projectUUID1, columnID1 := createTestProjectAndColumn(t, pool)
	projectUUID2, _ := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID1, creatorUUID)

	tag, _ := repo.Create(ctx, projectUUID1, "TestTag", 0x111111)
	require.NoError(t, repo.AddToTask(ctx, projectUUID1, AssignTagInput{TaskID: taskID, TagID: tag.ID}))

	// Пытаемся получить теги задачи через проект 2
	tags, err := repo.FindByTask(ctx, projectUUID2, taskID)
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestTagRepository_AddToTask_Success(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	tag, _ := repo.Create(ctx, projectUUID, "AddedTag", 0x555555)

	err := repo.AddToTask(ctx, projectUUID, AssignTagInput{TaskID: taskID, TagID: tag.ID})
	assert.NoError(t, err)

	tags, _ := repo.FindByTask(ctx, projectUUID, taskID)
	require.Len(t, tags, 1)
	assert.Equal(t, "AddedTag", tags[0].Name)
}

func TestTagRepository_AddToTask_Idempotent(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	tag, _ := repo.Create(ctx, projectUUID, "IdemTag", 0x666666)

	err1 := repo.AddToTask(ctx, projectUUID, AssignTagInput{TaskID: taskID, TagID: tag.ID})
	err2 := repo.AddToTask(ctx, projectUUID, AssignTagInput{TaskID: taskID, TagID: tag.ID})

	assert.NoError(t, err1)
	assert.NoError(t, err2) // Повторное добавление не должно вызывать ошибку

	tags, _ := repo.FindByTask(ctx, projectUUID, taskID)
	require.Len(t, tags, 1) // Тег должен остаться один
}

func TestTagRepository_AddToTask_InvalidRefs(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	// Несуществующий тег
	err := repo.AddToTask(ctx, projectUUID, AssignTagInput{TaskID: taskID, TagID: 99999})
	require.Error(t, err)

	// Несуществующая задача
	err = repo.AddToTask(ctx, projectUUID, AssignTagInput{TaskID: 99999, TagID: 99999})
	require.Error(t, err)
}

func TestTagRepository_RemoveFromTask_Success(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	tag, _ := repo.Create(ctx, projectUUID, "ToRemove", 0x777777)
	require.NoError(t, repo.AddToTask(ctx, projectUUID, AssignTagInput{TaskID: taskID, TagID: tag.ID}))

	err := repo.RemoveFromTask(ctx, projectUUID, taskID, tag.ID)
	assert.NoError(t, err)

	tags, _ := repo.FindByTask(ctx, projectUUID, taskID)
	assert.Empty(t, tags)
}

func TestTagRepository_RemoveFromTask_NotFound(t *testing.T) {
	repo, pool := setupTagRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	// Пытаемся убрать несуществующую связь
	err := repo.RemoveFromTask(ctx, projectUUID, taskID, 99999)
	assert.Error(t, err)
}
