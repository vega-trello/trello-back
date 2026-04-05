//go:build integration
// +build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "failed to connect to test database")

	err = pool.Ping(ctx)
	require.NoError(t, err, "cannot ping test database")

	return pool
}

func cleanTable(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, "TRUNCATE project RESTART IDENTITY CASCADE")
	if err != nil {
		t.Logf("warning: failed to clean table: %v", err)
	}
}

func setupRepo(t *testing.T) *ProjectRepository {
	t.Helper()
	pool := getTestPool(t)
	t.Cleanup(func() {
		cleanTable(t, pool)
		pool.Close()
	})
	return NewProjectRepository(pool)
}

func TestProjectRepository_Create_Success(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	project, err := repo.Create(ctx, "Test Project", "Description")
	require.NoError(t, err)
	require.NotNil(t, project)

	assert.NotEqual(t, uuid.Nil, project.UUID)
	assert.Equal(t, "Test Project", project.Title)
	assert.Equal(t, "Description", project.Description)
	assert.WithinDuration(t, time.Now(), project.CreatedAt, time.Second)
}

func TestProjectRepository_FindByID_Success(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, "Find Me", "Desc")
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, created.UUID)
	require.NoError(t, err)
	assert.Equal(t, created.UUID, found.UUID)
	assert.Equal(t, created.Title, found.Title)
	assert.Equal(t, created.Description, found.Description)
}

func TestProjectRepository_FindByID_NotFound(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestProjectRepository_Update_Success_Full(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, "Old Title", "Old Desc")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond) // Чтобы updated_at отличался

	title := "New Title"
	desc := "New Desc"
	updated, err := repo.Update(ctx, created.UUID, &title, &desc)
	require.NoError(t, err)

	assert.Equal(t, "New Title", updated.Title)
	assert.Equal(t, "New Desc", updated.Description)
	assert.True(t, updated.UpdatedAt.After(updated.CreatedAt))
}

func TestProjectRepository_Update_Partial_TitleOnly(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, "Keep This", "Change Me")
	require.NoError(t, err)

	newTitle := "Only Title Changed"
	updated, err := repo.Update(ctx, created.UUID, &newTitle, nil)
	require.NoError(t, err)

	assert.Equal(t, "Only Title Changed", updated.Title)
	assert.Equal(t, "Change Me", updated.Description) // Не изменилось
}

func TestProjectRepository_Update_Partial_DescOnly(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, "Keep Title", "Old Desc")
	require.NoError(t, err)

	newDesc := "Only Desc Changed"
	updated, err := repo.Update(ctx, created.UUID, nil, &newDesc)
	require.NoError(t, err)

	assert.Equal(t, "Keep Title", updated.Title)
	assert.Equal(t, "Only Desc Changed", updated.Description)
}

func TestProjectRepository_Update_NotFound(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	title := "Test"
	_, err := repo.Update(ctx, uuid.New(), &title, nil)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestProjectRepository_Delete_Success(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, "To Delete", "Desc")
	require.NoError(t, err)

	err = repo.Delete(ctx, created.UUID)
	assert.NoError(t, err)

	// Проверяем, что удалён
	_, err = repo.FindByID(ctx, created.UUID)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestProjectRepository_Delete_NotFound(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()

	err := repo.Delete(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrProjectNotFound)
}
