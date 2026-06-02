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

// setupTaskRepo инициализирует репозиторий и возвращает тестовые данные
func setupTaskRepo(t *testing.T) (*TaskRepository, *pgxpool.Pool, uuid.UUID, uuid.UUID) {
	t.Helper()
	pool := setupTestPool(t)
	repo := NewTaskRepository(pool)
	owner := createTestUser(t, pool, "task_owner", "pass123")
	projectUUID := createTestProject(t, pool, owner)
	return repo, pool, owner, projectUUID
}

func createTestStatus(t *testing.T, pool *pgxpool.Pool, projectUUID uuid.UUID, creatorUUID uuid.UUID, name string) int {
	t.Helper()
	var statusID int
	err := pool.QueryRow(context.Background(), `
		INSERT INTO project_status (project_uuid, name, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id
	`, projectUUID, name).Scan(&statusID)
	require.NoError(t, err)
	return statusID
}

func TestTaskRepository_Create_Success(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	task, err := repo.Create(ctx, projectUUID, columnID, nil, owner, "New Task", "Task description", nil, false, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, task)

	assert.NotZero(t, task.ID)
	assert.Equal(t, columnID, task.ColumnID)
	assert.Nil(t, task.StatusID)
	assert.Equal(t, "New Task", task.Title)
	assert.Equal(t, owner, task.CreatorUUID)
	assert.False(t, task.CreatedAt.IsZero())
	assert.False(t, task.UpdatedAt.IsZero())
	assert.False(t, task.Done)
	assert.Nil(t, task.Color)
}

func TestTaskRepository_Create_WithStatus_Success(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	// Создаём тестовый статус
	statusID := createTestStatus(t, pool, projectUUID, owner, "In Progress")

	// Создаём задачу с начальным статусом
	task, err := repo.Create(ctx, projectUUID, columnID, &statusID, owner, "Task with Status", "Desc", nil, false, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, task)

	assert.Equal(t, statusID, *task.StatusID)
	assert.Equal(t, "Task with Status", task.Title)
}

func TestTaskRepository_Create_WithColorAndDone_Success(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	color := "#FF5733"
	done := true

	task, err := repo.Create(ctx, projectUUID, columnID, nil, owner, "Colored Task", "Desc", &color, done, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, task)

	assert.Equal(t, "Colored Task", task.Title)
	assert.Equal(t, color, *task.Color)
	assert.True(t, task.Done)
}

func TestTaskRepository_Create_InvalidColumn(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()

	// Создаём колонку в ДРУГОМ проекте
	otherProject := createTestProject(t, pool, owner)
	otherColumnID := createTestColumn(t, pool, otherProject, 0)

	// Пытаемся создать задачу в projectUUID, но с columnID из otherProject
	_, err := repo.Create(ctx, projectUUID, otherColumnID, nil, owner, "Task", "Desc", nil, false, nil, nil)
	assert.ErrorIs(t, err, ErrInvalidColumn)
}

func TestTaskRepository_Create_InvalidStatus(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	// Создаём статус в ДРУГОМ проекте
	otherProject := createTestProject(t, pool, owner)
	otherStatusID := createTestStatus(t, pool, otherProject, owner, "Other Status")

	// Пытаемся создать задачу со статусом из чужого проекта
	_, err := repo.Create(ctx, projectUUID, columnID, &otherStatusID, owner, "Task", "Desc", nil, false, nil, nil)
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestTaskRepository_Create_AccessDenied(t *testing.T) {
	repo, pool, _, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	// Создаём пользователя, который НЕ является участником проекта
	outsider := createTestUser(t, pool, "outsider", "pass123")

	_, err := repo.Create(ctx, projectUUID, columnID, nil, outsider, "Secret Task", "Desc", nil, false, nil, nil)
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestTaskRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	_, _ = repo.Create(ctx, projectUUID, columnID, nil, owner, "Task 1", "Desc 1", nil, false, nil, nil)
	time.Sleep(10 * time.Millisecond)
	_, _ = repo.Create(ctx, projectUUID, columnID, nil, owner, "Task 2", "Desc 2", nil, false, nil, nil)

	tasks, err := repo.FindByProjectUUID(ctx, projectUUID, owner, nil)
	require.NoError(t, err)
	require.Len(t, tasks, 2)

	// Проверяем сортировку: новые задачи первыми (ORDER BY created_at DESC)
	assert.Equal(t, "Task 2", tasks[0].Title)
	assert.Equal(t, "Task 1", tasks[1].Title)
}

func TestTaskRepository_FindByProjectUUID_FilterArchived(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	// Создаём активную задачу
	_, _ = repo.Create(ctx, projectUUID, columnID, nil, owner, "Active", "Desc", nil, false, nil, nil)

	// Создаём и архивируем задачу (через прямой SQL)
	archivedTask, _ := repo.Create(ctx, projectUUID, columnID, nil, owner, "Archived", "Desc", nil, false, nil, nil)
	_, _ = pool.Exec(ctx, `UPDATE tasks SET archived_at = NOW() WHERE id = $1`, archivedTask.ID)

	activeTasks, err := repo.FindByProjectUUID(ctx, projectUUID, owner, boolPtr(false))
	require.NoError(t, err)
	require.Len(t, activeTasks, 1)
	assert.Equal(t, "Active", activeTasks[0].Title)

	// Запрашиваем только архивные
	archivedTasks, err := repo.FindByProjectUUID(ctx, projectUUID, owner, boolPtr(true))
	require.NoError(t, err)
	require.Len(t, archivedTasks, 1)
	assert.Equal(t, "Archived", archivedTasks[0].Title)
}

func TestTaskRepository_FindByID_Success(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	created, _ := repo.Create(ctx, projectUUID, columnID, nil, owner, "Find Me", "Desc", nil, false, nil, nil)

	task, err := repo.FindByID(ctx, projectUUID, created.ID, owner)
	require.NoError(t, err)
	assert.Equal(t, created.ID, task.ID)
	assert.Equal(t, "Find Me", task.Title)
	assert.Equal(t, columnID, task.ColumnID)
	assert.False(t, task.Done)
	assert.Nil(t, task.Color)
}

func TestTaskRepository_FindByID_NotFound(t *testing.T) {
	repo, _, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, projectUUID, 99999, owner)
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestTaskRepository_FindByID_AccessDenied(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	created, _ := repo.Create(ctx, projectUUID, columnID, nil, owner, "Private", "Desc", nil, false, nil, nil)
	outsider := createTestUser(t, pool, "outsider2", "pass123")

	_, err := repo.FindByID(ctx, projectUUID, created.ID, outsider)
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestTaskRepository_Update_Success(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	task, _ := repo.Create(ctx, projectUUID, columnID, nil, owner, "Old Title", "Old Desc", nil, false, nil, nil)
	oldUpdatedAt := task.UpdatedAt

	newTitle := "New Title"
	newDesc := "Updated description"

	updated, err := repo.Update(ctx, projectUUID, task.ID, owner, &newTitle, &newDesc, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "New Title", updated.Title)
	assert.Equal(t, "Updated description", updated.Description)
	assert.True(t, updated.UpdatedAt.After(oldUpdatedAt))
}

func TestTaskRepository_Update_WithStatus_Success(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	// Создаём два статуса
	statusA := createTestStatus(t, pool, projectUUID, owner, "Status A")
	statusB := createTestStatus(t, pool, projectUUID, owner, "Status B")

	// Создаём задачу со статусом A
	task, _ := repo.Create(ctx, projectUUID, columnID, &statusA, owner, "Task", "Desc", nil, false, nil, nil)
	assert.Equal(t, statusA, *task.StatusID)

	// Обновляем задачу, меняя статус на B
	updated, err := repo.Update(ctx, projectUUID, task.ID, owner, nil, nil, nil, nil, nil, nil, nil, &statusB, nil)
	require.NoError(t, err)

	assert.Equal(t, statusB, *updated.StatusID)
}

func TestTaskRepository_Update_WithColorAndDone_Success(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	task, _ := repo.Create(ctx, projectUUID, columnID, nil, owner, "Task", "Desc", nil, false, nil, nil)
	assert.Nil(t, task.Color)
	assert.False(t, task.Done)

	color := "#00FF00"
	done := true

	updated, err := repo.Update(ctx, projectUUID, task.ID, owner, nil, nil, nil, nil, &color, &done, nil, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, color, *updated.Color)
	assert.True(t, updated.Done)
}

func TestTaskRepository_Update_InvalidStatus(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	task, _ := repo.Create(ctx, projectUUID, columnID, nil, owner, "Task", "Desc", nil, false, nil, nil)

	// Создаём статус в другом проекте
	otherProject := createTestProject(t, pool, owner)
	otherStatusID := createTestStatus(t, pool, otherProject, owner, "Other")

	// Пытаемся обновить задачу, установив статус из чужого проекта
	_, err := repo.Update(ctx, projectUUID, task.ID, owner, nil, nil, nil, nil, nil, nil, nil, &otherStatusID, nil)
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestTaskRepository_Update_Archive(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	task, _ := repo.Create(ctx, projectUUID, columnID, nil, owner, "Task", "Desc", nil, false, nil, nil)
	assert.Nil(t, task.ArchivedAt)

	// Архивируем задачу
	archived := true
	updated, err := repo.Update(ctx, projectUUID, task.ID, owner, nil, nil, nil, nil, nil, nil, nil, nil, &archived)
	require.NoError(t, err)
	assert.NotNil(t, updated.ArchivedAt)

	// Разархивируем
	archived = false
	updated, err = repo.Update(ctx, projectUUID, task.ID, owner, nil, nil, nil, nil, nil, nil, nil, nil, &archived)
	require.NoError(t, err)
	assert.Nil(t, updated.ArchivedAt)
}

func TestTaskRepository_Delete_Success(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	task, _ := repo.Create(ctx, projectUUID, columnID, nil, owner, "To Delete", "Desc", nil, false, nil, nil)

	err := repo.Delete(ctx, projectUUID, task.ID, owner)
	require.NoError(t, err)

	_, err = repo.FindByID(ctx, projectUUID, task.ID, owner)
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestTaskRepository_Delete_AccessDenied(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	task, _ := repo.Create(ctx, projectUUID, columnID, nil, owner, "Protected", "Desc", nil, false, nil, nil)
	outsider := createTestUser(t, pool, "outsider3", "pass123")

	err := repo.Delete(ctx, projectUUID, task.ID, outsider)
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestTaskRepository_Move_Success(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	col1 := createTestColumn(t, pool, projectUUID, 0)
	col2 := createTestColumn(t, pool, projectUUID, 1)

	task, _ := repo.Create(ctx, projectUUID, col1, nil, owner, "Move Me", "Desc", nil, false, nil, nil)

	err := repo.Move(ctx, projectUUID, task.ID, col2, owner)
	require.NoError(t, err)

	updated, _ := repo.FindByID(ctx, projectUUID, task.ID, owner)
	assert.Equal(t, col2, updated.ColumnID)
}

func TestTaskRepository_Move_InvalidColumn(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	task, _ := repo.Create(ctx, projectUUID, columnID, nil, owner, "Task", "Desc", nil, false, nil, nil)

	// Создаём колонку в другом проекте
	otherProject := createTestProject(t, pool, owner)
	otherColumnID := createTestColumn(t, pool, otherProject, 0)

	// Пытаемся переместить задачу в колонку из чужого проекта
	err := repo.Move(ctx, projectUUID, task.ID, otherColumnID, owner)
	assert.ErrorIs(t, err, ErrInvalidColumn)
}

func TestTaskRepository_Archive_Success(t *testing.T) {
	repo, pool, owner, projectUUID := setupTaskRepo(t)
	ctx := context.Background()
	columnID := createTestColumn(t, pool, projectUUID, 0)

	task, _ := repo.Create(ctx, projectUUID, columnID, nil, owner, "Task", "Desc", nil, false, nil, nil)
	assert.Nil(t, task.ArchivedAt)

	// Архивируем через отдельный метод
	err := repo.Archive(ctx, projectUUID, task.ID, owner, true)
	require.NoError(t, err)

	// Проверяем, что задача заархивирована
	updated, _ := repo.FindByID(ctx, projectUUID, task.ID, owner)
	assert.NotNil(t, updated.ArchivedAt)

	// Разархивируем
	err = repo.Archive(ctx, projectUUID, task.ID, owner, false)
	require.NoError(t, err)

	updated, _ = repo.FindByID(ctx, projectUUID, task.ID, owner)
	assert.Nil(t, updated.ArchivedAt)
}
