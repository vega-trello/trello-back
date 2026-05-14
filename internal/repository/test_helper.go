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
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const (
	RoleOwner  = 1
	RoleAdmin  = 2
	RoleMember = 3
	RoleViewer = 4
)

func intPtr(i int) *int              { return &i }
func stringPtr(s string) *string     { return &s }
func boolPtr(b bool) *bool           { return &b }
func timePtr(t time.Time) *time.Time { return &t }

func ensureSystemRoles(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	roles := []struct {
		id          int
		name        string
		description string
	}{
		{RoleOwner, "Owner", "Project owner with full access"},
		{RoleAdmin, "Admin", "Project administrator"},
		{RoleMember, "Member", "Regular project member"},
		{RoleViewer, "Viewer", "Read-only access"},
	}

	for _, r := range roles {
		// Пробуем с created_at, если ошибка - пробуем без него
		_, err := pool.Exec(ctx, `
			INSERT INTO role (id, project_uuid, name, description)
			VALUES ($1, NULL, $2, $3,
			ON CONFLICT (id) DO NOTHING
		`, r.id, r.name, r.description)

		if err != nil {
			_, err = pool.Exec(ctx, `
				INSERT INTO role (id, project_uuid, name, description)
				VALUES ($1, NULL, $2, $3)
				ON CONFLICT (id) DO NOTHING
			`, r.id, r.name, r.description)
		}

		if err != nil {
			t.Logf("Warning: failed to ensure system role %d: %v", r.id, err)
		}
	}

	_, err := pool.Exec(ctx, `
		SELECT SETVAL(
			(SELECT pg_get_serial_sequence('role', 'id')),
			(SELECT COALESCE(MAX(id), 0) + 1 FROM role),
			false
		)
	`)
	if err != nil {
		t.Logf("Warning: failed to sync role sequence: %v", err)
	}
}

func setupTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:Pypochek2@localhost:5432/trega?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "failed to connect to test database")

	ensureSystemRoles(t, pool)

	t.Cleanup(func() {
		// Очищаем данные, но не трогаем системные роли
		_, _ = pool.Exec(ctx, `
			TRUNCATE 
				task_tag, task_assignee, tasks, project_column, project_status,
				project_member, tag, role_permission, permission,
				sso_user, manual_user, base_user, project 
			RESTART IDENTITY CASCADE;
		`)
		// Очищаем только роли проектов (не системные с project_uuid IS NULL)
		_, _ = pool.Exec(ctx, `DELETE FROM role WHERE project_uuid IS NOT NULL`)

		_, _ = pool.Exec(ctx, `
			SELECT SETVAL(
				(SELECT pg_get_serial_sequence('role', 'id')),
				(SELECT COALESCE(MAX(id), 4) + 1 FROM role),
				false
			)
		`)
	})
	return pool
}

func hashPassword(t *testing.T, password string) []byte {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	return hash
}

func createTestUser(t *testing.T, pool *pgxpool.Pool, username string, password string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	userUUID := uuid.New()
	hash := hashPassword(t, password)
	now := time.Now()

	_, err := pool.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, 'manual', $3, $3)
	`, userUUID, username, now)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO manual_user (user_uuid, password_hash)
		VALUES ($1, $2)
	`, userUUID, hash)
	require.NoError(t, err)

	return userUUID
}

func createTestProject(t *testing.T, pool *pgxpool.Pool, ownerUUID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	projectUUID := uuid.New()
	now := time.Now()

	_, err := pool.Exec(ctx, `
		INSERT INTO project (uuid, title, description, created_at, updated_at)
		VALUES ($1, 'Test Project', 'Description', $2, $2)
	`, projectUUID, now)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO project_member (project_uuid, user_uuid, role_id, joined_at)
		VALUES ($1, $2, 1, $3)
	`, projectUUID, ownerUUID, now)
	require.NoError(t, err)

	return projectUUID
}

func createTestColumn(t *testing.T, pool *pgxpool.Pool, projectUUID uuid.UUID, position int) int {
	t.Helper()
	ctx := context.Background()
	var colID int
	err := pool.QueryRow(ctx, `
		INSERT INTO project_column (project_uuid, name, position, created_at)
		VALUES ($1, 'Test Column', $2, NOW())
		RETURNING id
	`, projectUUID, position).Scan(&colID)
	require.NoError(t, err)
	return colID
}
