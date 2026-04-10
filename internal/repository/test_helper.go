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

func createTestBaseUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	u := uuid.New()
	// Генерируем уникальный username, чтобы не нарушать UNIQUE constraint
	username := "testuser_" + u.String()[:8]

	_, err := pool.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, u, username, "manual")

	require.NoError(t, err, "failed to create base_user")
	return u
}

func createTestManualUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	userUUID := createTestBaseUser(t, pool)

	_, err := pool.Exec(ctx, `
		INSERT INTO manual_user (user_uuid, password_hash)
		VALUES ($1, $2)
	`, userUUID, "fake_hash_for_tests")

	require.NoError(t, err, "failed to create manual_user")
	return userUUID
}

func createTestProject(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	u := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO project (uuid, title, description, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW())`, u, "Test Project", "Desc")
	require.NoError(t, err)
	return u
}

func cleanAllTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	_, _ = pool.Exec(ctx, "TRUNCATE project_member, project_column, project, base_user, task_assignee, tasks RESTART IDENTITY CASCADE")
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

	t.Cleanup(func() { pool.Close() })
	return pool
}

func ensureGlobalRoles(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		INSERT INTO role (id, name, description, project_uuid) VALUES 
			(1, 'owner', 'Project owner', NULL),
			(2, 'admin', 'Administrator', NULL),
			(3, 'member', 'Regular member', NULL),
			(4, 'viewer', 'Read-only access', NULL)
		ON CONFLICT (id) DO NOTHING
	`)
	require.NoError(t, err, "failed to ensure global roles")
}
