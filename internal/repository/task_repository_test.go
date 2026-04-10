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

func setupTaskRepo(t *testing.T) (*TaskRepository, *pgxpool.Pool) {
	t.Helper()
	pool := setupTestPool(t)
	repo := NewTaskRepository(pool)

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "TRUNCATE task_assignee, tasks RESTART IDENTITY CASCADE")
		cleanAllTables(t, pool)
	})

	return repo, pool
}

func createTestStatus(t *testing.T, pool *pgxpool.Pool, projectUUID uuid.UUID, id int, name string) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO project_status (id, project_uuid, name) 
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO NOTHING  -- ← Исправлено: id уже PRIMARY KEY
	`, id, projectUUID, name)
	require.NoError(t, err, "failed to create test status")
}

func createTestColumn(t *testing.T, pool *pgxpool.Pool, projectUUID uuid.UUID, position int, name string) int {
	t.Helper()
	ctx := context.Background()

	var columnID int
	err := pool.QueryRow(ctx, `
		INSERT INTO project_column (project_uuid, position, name, created_at) 
		VALUES ($1, $2, $3, NOW()) 
		RETURNING id
	`, projectUUID, position, name).Scan(&columnID)
	require.NoError(t, err, "failed to create test column")

	return columnID
}

func createTestProjectAndColumn(t *testing.T, pool *pgxpool.Pool) (uuid.UUID, int) {
	t.Helper()
	projectUUID := createTestProject(t, pool)
	columnID := createTestColumn(t, pool, projectUUID, 1, "Test Column")
	return projectUUID, columnID
}

func TestTaskRepository_Create_Success(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	input := CreateTaskInput{
		ProjectUUID: projectUUID,
		ColumnID:    columnID,
		Title:       "New Task",
		Description: "Task description",
		CreatorUUID: creatorUUID,
	}

	task, err := repo.Create(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, task)

	assert.Equal(t, columnID, task.ColumnID)
	assert.Equal(t, "New Task", task.Title)
	assert.Equal(t, "Task description", task.Description)
	assert.Equal(t, creatorUUID, task.CreatorUUID)
	assert.Nil(t, task.StatusID)
	assert.Nil(t, task.DeletedAt)
	assert.Nil(t, task.ArchivedAt)
	assert.WithinDuration(t, time.Now(), task.CreatedAt, 2*time.Second)
	assert.WithinDuration(t, time.Now(), task.UpdatedAt, 2*time.Second)
}

func TestTaskRepository_Create_WithOptionalFields(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	createTestStatus(t, pool, projectUUID, 2, "In Progress")

	statusID := 2
	startDate := time.Now().Add(24 * time.Hour)
	endDate := time.Now().Add(48 * time.Hour)

	input := CreateTaskInput{
		ProjectUUID: projectUUID,
		ColumnID:    columnID,
		Title:       "Complex Task",
		CreatorUUID: creatorUUID,
		StatusID:    &statusID,
		StartDate:   &startDate,
		EndDate:     &endDate,
	}

	task, err := repo.Create(ctx, input)
	require.NoError(t, err)

	assert.Equal(t, statusID, *task.StatusID)
	assert.WithinDuration(t, startDate, *task.StartDate, 2*time.Second)
	assert.WithinDuration(t, endDate, *task.EndDate, 2*time.Second)
}

func TestTaskRepository_Create_WithStatus(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	createTestStatus(t, pool, projectUUID, 3, "Done")

	statusID := 3
	input := CreateTaskInput{
		ProjectUUID: projectUUID,
		ColumnID:    columnID,
		Title:       "Task with Status",
		CreatorUUID: creatorUUID,
		StatusID:    &statusID,
	}

	task, err := repo.Create(ctx, input)
	require.NoError(t, err)

	assert.Equal(t, 3, *task.StatusID)
	assert.Equal(t, "Task with Status", task.Title)
}

func TestTaskRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	createTestStatus(t, pool, projectUUID, 1, "To Do")
	statusID := 1

	_, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Task 1", CreatorUUID: creatorUUID, StatusID: &statusID})
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	_, err = repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Task 2", CreatorUUID: creatorUUID, StatusID: &statusID})
	require.NoError(t, err)

	tasks, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, tasks, 2)

	assert.Equal(t, "Task 1", tasks[0].Title)
	assert.Equal(t, "Task 2", tasks[1].Title)
	assert.True(t, tasks[0].CreatedAt.Before(tasks[1].CreatedAt))
}

func TestTaskRepository_FindByProjectUUID_ExcludesDeleted(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	_, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Active", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	task2, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Deleted", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	err = repo.Delete(ctx, task2.ID)
	require.NoError(t, err)

	tasks, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	assert.Equal(t, "Active", tasks[0].Title)
}

func TestTaskRepository_FindByProjectUUID_Empty(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	tasks, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestTaskRepository_FindByID_Success(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	created, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Find Me", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	task, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, task.ID)
	assert.Equal(t, "Find Me", task.Title)
}

func TestTaskRepository_FindByID_WithStatus(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	createTestStatus(t, pool, projectUUID, 2, "In Progress")
	statusID := 2

	created, err := repo.Create(ctx, CreateTaskInput{
		ProjectUUID: projectUUID,
		ColumnID:    columnID,
		Title:       "Task with Status",
		CreatorUUID: creatorUUID,
		StatusID:    &statusID,
	})
	require.NoError(t, err)

	task, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, task.ID)
	assert.Equal(t, "Task with Status", task.Title)
	assert.NotNil(t, task.StatusID)
	assert.Equal(t, 2, *task.StatusID)
}

func TestTaskRepository_FindByID_ReturnsDeleted(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	created, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Deleted Task", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)

	task, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, task.DeletedAt)
	assert.Equal(t, "Deleted Task", task.Title)
}

func TestTaskRepository_FindByID_NotFound(t *testing.T) {
	repo, _ := setupTaskRepo(t)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, 99999)
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestTaskRepository_Update_Success(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	createTestStatus(t, pool, projectUUID, 1, "To Do")
	createTestStatus(t, pool, projectUUID, 3, "Done")

	task, err := repo.Create(ctx, CreateTaskInput{
		ProjectUUID: projectUUID,
		ColumnID:    columnID,
		Title:       "Old Title",
		CreatorUUID: creatorUUID,
		StatusID:    func() *int { i := 1; return &i }(),
	})
	require.NoError(t, err)

	originalUpdated := task.UpdatedAt

	time.Sleep(10 * time.Millisecond)

	newTitle := "New Title"
	newDesc := "Updated description"
	statusID := 3
	updates := TaskUpdates{
		Title:       &newTitle,
		Description: &newDesc,
		StatusID:    &statusID,
	}

	updated, err := repo.Update(ctx, task.ID, updates)
	require.NoError(t, err)

	assert.Equal(t, "New Title", updated.Title)
	assert.Equal(t, "Updated description", updated.Description)
	assert.Equal(t, 3, *updated.StatusID)
	assert.True(t, updated.UpdatedAt.After(originalUpdated))
}

func TestTaskRepository_Update_Partial(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	task, err := repo.Create(ctx, CreateTaskInput{
		ProjectUUID: projectUUID,
		ColumnID:    columnID,
		Title:       "Keep This",
		Description: "Keep This Too",
		CreatorUUID: creatorUUID,
	})
	require.NoError(t, err)

	newTitle := "Only Title Changed"
	updates := TaskUpdates{Title: &newTitle}

	updated, err := repo.Update(ctx, task.ID, updates)
	require.NoError(t, err)

	assert.Equal(t, "Only Title Changed", updated.Title)
	assert.Equal(t, "Keep This Too", updated.Description)
}

func TestTaskRepository_Update_ChangeStatusOnly(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	createTestStatus(t, pool, projectUUID, 1, "To Do")
	createTestStatus(t, pool, projectUUID, 2, "In Progress")

	task, err := repo.Create(ctx, CreateTaskInput{
		ProjectUUID: projectUUID,
		ColumnID:    columnID,
		Title:       "Task",
		CreatorUUID: creatorUUID,
		StatusID:    func() *int { i := 1; return &i }(),
	})
	require.NoError(t, err)

	newStatusID := 2
	updates := TaskUpdates{StatusID: &newStatusID}

	updated, err := repo.Update(ctx, task.ID, updates)
	require.NoError(t, err)

	assert.Equal(t, "Task", updated.Title)
	assert.Equal(t, 2, *updated.StatusID)
}

func TestTaskRepository_Update_ArchiveAndUnarchive(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	task, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Archive Test", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	assert.Nil(t, task.ArchivedAt)

	// Архивируем
	now := time.Now()
	updates := TaskUpdates{ArchivedAt: &now}
	updated, err := repo.Update(ctx, task.ID, updates)
	require.NoError(t, err)
	require.NotNil(t, updated.ArchivedAt)

	zeroTime := time.Time{}
	updates = TaskUpdates{ArchivedAt: &zeroTime}
	updated, err = repo.Update(ctx, task.ID, updates)
	require.NoError(t, err)
	assert.Nil(t, updated.ArchivedAt)
}

func TestTaskRepository_Update_NotFound(t *testing.T) {
	repo, _ := setupTaskRepo(t)
	ctx := context.Background()

	title := "test"
	_, err := repo.Update(ctx, 99999, TaskUpdates{Title: &title})
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestTaskRepository_Delete_Success(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	task, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "To Delete", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	assert.Nil(t, task.DeletedAt)

	err = repo.Delete(ctx, task.ID)
	require.NoError(t, err)

	deletedTask, err := repo.FindByID(ctx, task.ID)
	require.NoError(t, err)
	require.NotNil(t, deletedTask.DeletedAt)
	assert.WithinDuration(t, time.Now(), *deletedTask.DeletedAt, 2*time.Second)
}

func TestTaskRepository_Delete_AlreadyDeleted(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	task, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Task", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	err = repo.Delete(ctx, task.ID)
	require.NoError(t, err)

	err = repo.Delete(ctx, task.ID)
	assert.ErrorIs(t, err, ErrTaskDeleted)
}

func TestTaskRepository_Delete_NotFound(t *testing.T) {
	repo, _ := setupTaskRepo(t)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestTaskRepository_Restore_Success(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	task, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Restore Me", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	err = repo.Delete(ctx, task.ID)
	require.NoError(t, err)

	err = repo.Restore(ctx, task.ID)
	require.NoError(t, err)

	restored, err := repo.FindByID(ctx, task.ID)
	require.NoError(t, err)
	assert.Nil(t, restored.DeletedAt)
}

func TestTaskRepository_Restore_AlreadyActive(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	task, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Task", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	err = repo.Restore(ctx, task.ID)
	assert.NoError(t, err)
}

func TestTaskRepository_AddAssignee_Success(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	assigneeUUID := createTestBaseUser(t, pool)

	task, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Task", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	err = repo.AddAssignee(ctx, task.ID, assigneeUUID)
	require.NoError(t, err)

	assignees, err := repo.GetAssignees(ctx, task.ID)
	require.NoError(t, err)
	require.Len(t, assignees, 1)
	assert.Equal(t, assigneeUUID, assignees[0])
}

func TestTaskRepository_AddAssignee_Idempotent(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	assigneeUUID := createTestBaseUser(t, pool)

	task, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Task", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	err = repo.AddAssignee(ctx, task.ID, assigneeUUID)
	require.NoError(t, err)

	err = repo.AddAssignee(ctx, task.ID, assigneeUUID)
	require.NoError(t, err)

	assignees, err := repo.GetAssignees(ctx, task.ID)
	require.NoError(t, err)
	require.Len(t, assignees, 1)
}

func TestTaskRepository_RemoveAssignee_Success(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	assigneeUUID := createTestBaseUser(t, pool)

	task, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Task", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	err = repo.AddAssignee(ctx, task.ID, assigneeUUID)
	require.NoError(t, err)

	err = repo.RemoveAssignee(ctx, task.ID, assigneeUUID)
	require.NoError(t, err)

	assignees, err := repo.GetAssignees(ctx, task.ID)
	require.NoError(t, err)
	assert.Empty(t, assignees)
}

func TestTaskRepository_RemoveAssignee_NotFound(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	task, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Task", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	err = repo.RemoveAssignee(ctx, task.ID, uuid.New())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "assignee not found")
}

func TestTaskRepository_GetAssignees_Empty(t *testing.T) {
	repo, pool := setupTaskRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	task, err := repo.Create(ctx, CreateTaskInput{ProjectUUID: projectUUID, ColumnID: columnID, Title: "Task", CreatorUUID: creatorUUID})
	require.NoError(t, err)

	assignees, err := repo.GetAssignees(ctx, task.ID)
	require.NoError(t, err)
	assert.Empty(t, assignees)
}
