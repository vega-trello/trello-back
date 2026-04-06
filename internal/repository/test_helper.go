// internal/repository/test_helpers.go
//go:build integration
// +build integration

package repository

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func createTestProject(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	projectUUID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO project (uuid, title, description, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, projectUUID, "Test Project", "Auto-created for integration tests")
	require.NoError(t, err, "failed to create test project")

	return projectUUID
}

func cleanAllTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	_, _ = pool.Exec(ctx, "TRUNCATE project_column, project RESTART IDENTITY CASCADE")
}

func setupTestPool(t *testing.T) *pgxpool.Pool {
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

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}
