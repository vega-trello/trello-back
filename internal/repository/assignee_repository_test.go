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

func setupAssigneeRepo(t *testing.T) (*AssigneeRepository, *pgxpool.Pool) {
	t.Helper()
	pool := setupTestPool(t)
	repo := NewAssigneeRepository(pool)

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "TRUNCATE task_assignee, tasks RESTART IDENTITY CASCADE")
		cleanAllTables(t, pool)
	})

	return repo, pool
}

func createTestTask(t *testing.T, pool *pgxpool.Pool, columnID int, creatorUUID uuid.UUID) int {
	t.Helper()
	ctx := context.Background()

	var taskID int
	err := pool.QueryRow(ctx, `
		INSERT INTO tasks (column_id, creator_uuid, title, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id
	`, columnID, creatorUUID, "Test Task").Scan(&taskID)
	require.NoError(t, err, "failed to create test task")

	return taskID
}

func TestAssigneeRepository_FindByTask_Success(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	user1 := createTestBaseUser(t, pool)
	user2 := createTestBaseUser(t, pool)
	user3 := createTestBaseUser(t, pool)

	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user1}))
	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user2}))
	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user3}))

	assignees, err := repo.FindByTask(ctx, projectUUID, taskID)
	require.NoError(t, err)
	require.Len(t, assignees, 3)

	uuids := make([]uuid.UUID, len(assignees))
	for i, a := range assignees {
		uuids[i] = a.UUID
	}

	assert.Contains(t, uuids, user1)
	assert.Contains(t, uuids, user2)
	assert.Contains(t, uuids, user3)
}

func TestAssigneeRepository_FindByTask_Empty(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	assignees, err := repo.FindByTask(ctx, projectUUID, taskID)
	require.NoError(t, err)
	assert.Empty(t, assignees)
}

func TestAssigneeRepository_FindByTask_TaskNotFound(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	assignees, err := repo.FindByTask(ctx, projectUUID, 99999)
	require.NoError(t, err)
	assert.Empty(t, assignees)
}

func TestAssigneeRepository_FindByTask_WrongProject(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()

	projectUUID1, columnID1 := createTestProjectAndColumn(t, pool)
	projectUUID2, _ := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	taskID := createTestTask(t, pool, columnID1, creatorUUID)
	user := createTestBaseUser(t, pool)
	require.NoError(t, repo.AddToTask(ctx, projectUUID1, taskID, AssignUserInput{UserUUID: user}))

	assignees, err := repo.FindByTask(ctx, projectUUID2, taskID)
	require.NoError(t, err)
	assert.Empty(t, assignees)
}

func TestAssigneeRepository_AddToTask_Success(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	user := createTestBaseUser(t, pool)

	err := repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user})
	assert.NoError(t, err)

	assignees, _ := repo.FindByTask(ctx, projectUUID, taskID)
	require.Len(t, assignees, 1)
	assert.Equal(t, user, assignees[0].UUID)
}

func TestAssigneeRepository_AddToTask_Idempotent(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	user := createTestBaseUser(t, pool)

	err1 := repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user})
	err2 := repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user})

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	assignees, _ := repo.FindByTask(ctx, projectUUID, taskID)
	require.Len(t, assignees, 1)
	assert.Equal(t, user, assignees[0].UUID)
}

func TestAssigneeRepository_AddToTask_MultipleUsers(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	user1 := createTestBaseUser(t, pool)
	user2 := createTestBaseUser(t, pool)
	user3 := createTestBaseUser(t, pool)

	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user1}))
	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user2}))
	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user3}))

	assignees, _ := repo.FindByTask(ctx, projectUUID, taskID)
	require.Len(t, assignees, 3)
}

func TestAssigneeRepository_AddToTask_TaskNotFound(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)
	user := createTestBaseUser(t, pool)
	err := repo.AddToTask(ctx, projectUUID, 99999, AssignUserInput{UserUUID: user})
	require.Error(t, err)
}

func TestAssigneeRepository_AddToTask_WrongProject(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()

	_, columnID1 := createTestProjectAndColumn(t, pool)
	projectUUID2, _ := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	taskID := createTestTask(t, pool, columnID1, creatorUUID)
	user := createTestBaseUser(t, pool)
	err := repo.AddToTask(ctx, projectUUID2, taskID, AssignUserInput{UserUUID: user})
	require.Error(t, err)
}

func TestAssigneeRepository_AddToTask_SameUserDifferentTasks(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	taskID1 := createTestTask(t, pool, columnID, creatorUUID)
	taskID2 := createTestTask(t, pool, columnID, creatorUUID)

	user := createTestBaseUser(t, pool)

	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID1, AssignUserInput{UserUUID: user}))
	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID2, AssignUserInput{UserUUID: user}))

	assignees1, _ := repo.FindByTask(ctx, projectUUID, taskID1)
	assignees2, _ := repo.FindByTask(ctx, projectUUID, taskID2)

	require.Len(t, assignees1, 1)
	require.Len(t, assignees2, 1)
	assert.Equal(t, user, assignees1[0].UUID)
	assert.Equal(t, user, assignees2[0].UUID)
}

func TestAssigneeRepository_RemoveFromTask_Success(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	user := createTestBaseUser(t, pool)
	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user}))

	err := repo.RemoveFromTask(ctx, projectUUID, taskID, user)
	assert.NoError(t, err)

	assignees, _ := repo.FindByTask(ctx, projectUUID, taskID)
	assert.Empty(t, assignees)
}

func TestAssigneeRepository_RemoveFromTask_NotFound(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	user := createTestBaseUser(t, pool)

	err := repo.RemoveFromTask(ctx, projectUUID, taskID, user)
	assert.ErrorIs(t, err, ErrAssigneeNotFound)
}

func TestAssigneeRepository_RemoveFromTask_WrongProject(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()

	projectUUID_A, colA := createTestProjectAndColumn(t, pool)
	projectUUID_B, _ := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)

	taskID := createTestTask(t, pool, colA, creatorUUID)
	user := createTestBaseUser(t, pool)

	require.NoError(t, repo.AddToTask(ctx, projectUUID_A, taskID, AssignUserInput{UserUUID: user}))

	err := repo.RemoveFromTask(ctx, projectUUID_B, taskID, user)
	assert.ErrorIs(t, err, ErrAssigneeNotFound)
}

func TestAssigneeRepository_RemoveFromTask_LeavesOthersIntact(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	user1 := createTestBaseUser(t, pool)
	user2 := createTestBaseUser(t, pool)
	user3 := createTestBaseUser(t, pool)

	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user1}))
	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user2}))
	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user3}))

	err := repo.RemoveFromTask(ctx, projectUUID, taskID, user2)
	require.NoError(t, err)

	assignees, _ := repo.FindByTask(ctx, projectUUID, taskID)
	require.Len(t, assignees, 2)

	for _, a := range assignees {
		assert.NotEqual(t, user2, a.UUID)
	}
}

func TestAssigneeRepository_FullWorkflow(t *testing.T) {
	repo, pool := setupAssigneeRepo(t)
	ctx := context.Background()
	projectUUID, columnID := createTestProjectAndColumn(t, pool)
	creatorUUID := createTestBaseUser(t, pool)
	taskID := createTestTask(t, pool, columnID, creatorUUID)

	user1 := createTestBaseUser(t, pool)
	user2 := createTestBaseUser(t, pool)
	user3 := createTestBaseUser(t, pool)

	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user1}))
	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user2}))
	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user3}))

	assignees, err := repo.FindByTask(ctx, projectUUID, taskID)
	require.NoError(t, err)
	require.Len(t, assignees, 3)

	err = repo.RemoveFromTask(ctx, projectUUID, taskID, user2)
	require.NoError(t, err)

	assignees, err = repo.FindByTask(ctx, projectUUID, taskID)
	require.NoError(t, err)
	require.Len(t, assignees, 2)

	foundUser1 := false
	foundUser3 := false
	for _, a := range assignees {
		if a.UUID == user1 {
			foundUser1 = true
		}
		if a.UUID == user3 {
			foundUser3 = true
		}
	}
	assert.True(t, foundUser1, "User1 should remain")
	assert.True(t, foundUser3, "User3 should remain")

	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user2}))
	require.NoError(t, repo.AddToTask(ctx, projectUUID, taskID, AssignUserInput{UserUUID: user2}))

	assignees, err = repo.FindByTask(ctx, projectUUID, taskID)
	require.NoError(t, err)
	require.Len(t, assignees, 3)
}
