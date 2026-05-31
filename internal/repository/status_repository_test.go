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

func setupStatusRepo(t *testing.T) (*StatusRepository, *pgxpool.Pool, uuid.UUID, uuid.UUID) {
	t.Helper()
	pool := setupTestPool(t)
	repo := NewStatusRepository(pool)
	owner := createTestUser(t, pool, "status_owner", "pass123")
	projectUUID := createTestProject(t, pool, owner)
	return repo, pool, owner, projectUUID
}

func TestStatusRepository_Create_Success(t *testing.T) {
	repo, _, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	status, err := repo.Create(ctx, projectUUID, "In Progress", owner)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.NotZero(t, status.ID)
	assert.Equal(t, "In Progress", status.Name)
	assert.Equal(t, projectUUID, status.ProjectUUID)
	assert.False(t, status.CreatedAt.IsZero())
}

func TestStatusRepository_Create_DuplicateName(t *testing.T) {
	repo, _, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, projectUUID, "Duplicate", owner)
	require.NoError(t, err)

	_, err = repo.Create(ctx, projectUUID, "Duplicate", owner)
	assert.ErrorIs(t, err, ErrStatusAlreadyExists)
}

func TestStatusRepository_Create_AccessDenied(t *testing.T) {
	repo, pool, _, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	outsider := createTestUser(t, pool, "outsider", "pass123")
	_, err := repo.Create(ctx, projectUUID, "Secret", outsider)
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestStatusRepository_FindByProject_Success(t *testing.T) {
	repo, _, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	_, _ = repo.Create(ctx, projectUUID, "To Do", owner)
	_, _ = repo.Create(ctx, projectUUID, "In Progress", owner)
	_, _ = repo.Create(ctx, projectUUID, "Done", owner)

	statuses, err := repo.FindByProject(ctx, projectUUID, owner)
	require.NoError(t, err)
	require.Len(t, statuses, 3)

	for _, s := range statuses {
		assert.Equal(t, projectUUID, s.ProjectUUID)
	}
}

func TestStatusRepository_FindByProject_Empty_Success(t *testing.T) {
	repo, pool, owner, _ := setupStatusRepo(t)
	ctx := context.Background()

	emptyProject := createTestProject(t, pool, owner)

	statuses, err := repo.FindByProject(ctx, emptyProject, owner)
	require.NoError(t, err)

	assert.Empty(t, statuses)
	assert.NotNil(t, statuses)
}

func TestStatusRepository_FindByProject_AccessDenied(t *testing.T) {
	repo, pool, _, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	outsider := createTestUser(t, pool, "outsider2", "pass123")
	_, err := repo.FindByProject(ctx, projectUUID, outsider)
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestStatusRepository_FindByID_Success(t *testing.T) {
	repo, _, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, projectUUID, "Find Me", owner)
	found, err := repo.FindByID(ctx, projectUUID, created.ID, owner)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Find Me", found.Name)
	assert.Equal(t, projectUUID, found.ProjectUUID)
}

func TestStatusRepository_FindByID_NotFound(t *testing.T) {
	repo, _, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, projectUUID, 99999, owner)
	assert.ErrorIs(t, err, ErrStatusNotFound)
}

func TestStatusRepository_FindByID_WrongProject(t *testing.T) {
	repo, pool, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, projectUUID, "Test", owner)
	otherProject := createTestProject(t, pool, owner)

	_, err := repo.FindByID(ctx, otherProject, created.ID, owner)
	assert.ErrorIs(t, err, ErrStatusNotFound)
}

func TestStatusRepository_Update_Success(t *testing.T) {
	repo, _, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, projectUUID, "Old Name", owner)
	updated, err := repo.Update(ctx, projectUUID, created.ID, "New Name", owner)
	require.NoError(t, err)

	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, created.ID, updated.ID)
}

func TestStatusRepository_Update_DuplicateName(t *testing.T) {
	repo, _, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	_, _ = repo.Create(ctx, projectUUID, "Status A", owner)
	statusB, _ := repo.Create(ctx, projectUUID, "Status B", owner)

	_, err := repo.Update(ctx, projectUUID, statusB.ID, "Status A", owner)
	assert.ErrorIs(t, err, ErrStatusAlreadyExists)
}

func TestStatusRepository_Update_SameNameAllowed(t *testing.T) {
	repo, _, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, projectUUID, "Same Name", owner)
	_, err := repo.Update(ctx, projectUUID, created.ID, "Same Name", owner)
	assert.NoError(t, err)
}

func TestStatusRepository_Update_AccessDenied(t *testing.T) {
	repo, pool, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	outsider := createTestUser(t, pool, "outsider3", "pass123")
	created, _ := repo.Create(ctx, projectUUID, "Test", owner)

	_, err := repo.Update(ctx, projectUUID, created.ID, "Hacked", outsider)
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestStatusRepository_Delete_Success(t *testing.T) {
	repo, _, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, projectUUID, "ToDelete", owner)
	err := repo.Delete(ctx, projectUUID, created.ID, owner)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, projectUUID, created.ID, owner)
	assert.ErrorIs(t, err, ErrStatusNotFound)
}

func TestStatusRepository_Delete_WithActiveTasks(t *testing.T) {
	repo, pool, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	columnID := createTestColumn(t, pool, projectUUID, 0)

	status, _ := repo.Create(ctx, projectUUID, "With Tasks", owner)

	_, err := pool.Exec(ctx, `
		INSERT INTO tasks (column_id, status_id, title, creator_uuid, created_at, updated_at)
		VALUES ($1, $2, 'Active Task', $3, NOW(), NOW())
	`, columnID, status.ID, owner)
	require.NoError(t, err)

	err = repo.Delete(ctx, projectUUID, status.ID, owner)
	assert.ErrorIs(t, err, ErrStatusHasActiveTasks)

	found, err := repo.FindByID(ctx, projectUUID, status.ID, owner)
	assert.NoError(t, err)
	assert.Equal(t, status.ID, found.ID)
}

func TestStatusRepository_Delete_WithArchivedTasksAllowed(t *testing.T) {
	repo, pool, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	columnID := createTestColumn(t, pool, projectUUID, 0)

	status, _ := repo.Create(ctx, projectUUID, "Archived Only", owner)

	_, err := pool.Exec(ctx, `
		INSERT INTO tasks (column_id, status_id, title, creator_uuid, created_at, updated_at, archived_at)
		VALUES ($1, $2, 'Old Task', $3, NOW(), NOW(), NOW())
	`, columnID, status.ID, owner)
	require.NoError(t, err)

	err = repo.Delete(ctx, projectUUID, status.ID, owner)
	assert.NoError(t, err)
}

func TestStatusRepository_Delete_AccessDenied(t *testing.T) {
	repo, pool, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	outsider := createTestUser(t, pool, "outsider4", "pass123")
	created, _ := repo.Create(ctx, projectUUID, "Protected", owner)

	err := repo.Delete(ctx, projectUUID, created.ID, outsider)
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestStatusRepository_Delete_NotFound(t *testing.T) {
	repo, _, owner, projectUUID := setupStatusRepo(t)
	ctx := context.Background()

	err := repo.Delete(ctx, projectUUID, 99999, owner)
	assert.ErrorIs(t, err, ErrStatusNotFound)
}
