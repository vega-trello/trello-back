//go:build integration
// +build integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssigneeRepository_Add_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAssigneeRepository(pool)
	ctx := context.Background()

	owner := createTestUser(t, pool, "owner", "pass123")
	member := createTestUser(t, pool, "member", "pass123")
	projectUUID := createTestProject(t, pool, owner)
	columnID := createTestColumn(t, pool, projectUUID, 0)

	var taskID int
	err := pool.QueryRow(ctx, `
		INSERT INTO tasks (column_id, creator_uuid, title, description, created_at, updated_at)
		VALUES ($1, $2, 'Test Task', 'Desc', NOW(), NOW())
		RETURNING id
	`, columnID, owner).Scan(&taskID)
	require.NoError(t, err)

	// Добавляем member в проект, чтобы можно было назначать
	_, err = pool.Exec(ctx, `
		INSERT INTO project_member (project_uuid, user_uuid, role_id, joined_at)
		VALUES ($1, $2, 2, NOW())
	`, projectUUID, member)
	require.NoError(t, err)

	// Вызываем метод Add
	err = repo.Add(ctx, projectUUID, taskID, member, owner)
	assert.NoError(t, err)

	assignees, err := repo.FindByTask(ctx, projectUUID, taskID, owner)
	require.NoError(t, err)
	require.Len(t, assignees, 1)
	assert.Equal(t, member, assignees[0].UserUUID)
}

func TestAssigneeRepository_Remove_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAssigneeRepository(pool)
	ctx := context.Background()

	owner := createTestUser(t, pool, "owner2", "pass123")
	assignee := createTestUser(t, pool, "assignee", "pass123")
	projectUUID := createTestProject(t, pool, owner)
	columnID := createTestColumn(t, pool, projectUUID, 0)

	var taskID int
	err := pool.QueryRow(ctx, `
		INSERT INTO tasks (column_id, creator_uuid, title, created_at, updated_at)
		VALUES ($1, $2, 'Task', NOW(), NOW()) RETURNING id
	`, columnID, owner).Scan(&taskID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO project_member (project_uuid, user_uuid, role_id, joined_at)
		VALUES ($1, $2, 2, NOW())
	`, projectUUID, assignee)
	require.NoError(t, err)

	_ = repo.Add(ctx, projectUUID, taskID, assignee, owner)
	err = repo.Remove(ctx, projectUUID, taskID, assignee, owner)
	assert.NoError(t, err)

	assignees, err := repo.FindByTask(ctx, projectUUID, taskID, owner)
	require.NoError(t, err)
	assert.Empty(t, assignees)
}

func TestAssigneeRepository_Add_AlreadyAssigned(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAssigneeRepository(pool)
	ctx := context.Background()

	owner := createTestUser(t, pool, "owner3", "pass123")
	member := createTestUser(t, pool, "member2", "pass123")
	projectUUID := createTestProject(t, pool, owner)
	columnID := createTestColumn(t, pool, projectUUID, 0)

	var taskID int
	err := pool.QueryRow(ctx, `
		INSERT INTO tasks (column_id, creator_uuid, title, created_at, updated_at)
		VALUES ($1, $2, 'Task', NOW(), NOW()) RETURNING id
	`, columnID, owner).Scan(&taskID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO project_member (project_uuid, user_uuid, role_id, joined_at)
		VALUES ($1, $2, 2, NOW())
	`, projectUUID, member)
	require.NoError(t, err)

	// Первое добавление - успешно
	err = repo.Add(ctx, projectUUID, taskID, member, owner)
	require.NoError(t, err)

	// Повторное добавление - ошибка
	err = repo.Add(ctx, projectUUID, taskID, member, owner)
	assert.Equal(t, ErrAlreadyAssigned, err)
}

func TestAssigneeRepository_Add_UserNotFound(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAssigneeRepository(pool)
	ctx := context.Background()

	owner := createTestUser(t, pool, "owner4", "pass123")
	projectUUID := createTestProject(t, pool, owner)
	columnID := createTestColumn(t, pool, projectUUID, 0)

	var taskID int
	err := pool.QueryRow(ctx, `
		INSERT INTO tasks (column_id, creator_uuid, title, created_at, updated_at)
		VALUES ($1, $2, 'Task', NOW(), NOW()) RETURNING id
	`, columnID, owner).Scan(&taskID)
	require.NoError(t, err)

	// Пытаемся назначить несуществующего пользователя
	fakeUUID := uuid.New()
	err = repo.Add(ctx, projectUUID, taskID, fakeUUID, owner)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestAssigneeRepository_Remove_NotFound(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAssigneeRepository(pool)
	ctx := context.Background()

	owner := createTestUser(t, pool, "owner5", "pass123")
	assignee := createTestUser(t, pool, "assignee2", "pass123")
	projectUUID := createTestProject(t, pool, owner)
	columnID := createTestColumn(t, pool, projectUUID, 0)

	var taskID int
	err := pool.QueryRow(ctx, `
		INSERT INTO tasks (column_id, creator_uuid, title, created_at, updated_at)
		VALUES ($1, $2, 'Task', NOW(), NOW()) RETURNING id
	`, columnID, owner).Scan(&taskID)
	require.NoError(t, err)

	err = repo.Remove(ctx, projectUUID, taskID, assignee, owner)
	assert.Equal(t, ErrAssigneeNotFound, err)
}

func TestAssigneeRepository_Add_AccessDenied(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAssigneeRepository(pool)
	ctx := context.Background()

	owner := createTestUser(t, pool, "owner6", "pass123")
	outsider := createTestUser(t, pool, "outsider", "pass123")
	projectUUID := createTestProject(t, pool, owner)
	columnID := createTestColumn(t, pool, projectUUID, 0)

	var taskID int
	err := pool.QueryRow(ctx, `
		INSERT INTO tasks (column_id, creator_uuid, title, created_at, updated_at)
		VALUES ($1, $2, 'Task', NOW(), NOW()) RETURNING id
	`, columnID, owner).Scan(&taskID)
	require.NoError(t, err)

	err = repo.Add(ctx, projectUUID, taskID, outsider, outsider)
	assert.Equal(t, ErrAccessDenied, err)
}
