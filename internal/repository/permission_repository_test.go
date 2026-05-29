//go:build integration
// +build integration

package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vega-trello/trello-back/internal/model"
)

func truncatePermissions(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `TRUNCATE permission RESTART IDENTITY CASCADE`)
	require.NoError(t, err, "failed to truncate permission table")
}

func seedPermissions(t *testing.T, pool *pgxpool.Pool) []*model.Permission {
	t.Helper()
	ctx := context.Background()

	permissions := []*model.Permission{
		{ID: 1, Name: "view_project", Description: "Read-only access to project data"},
		{ID: 2, Name: "manage_project", Description: "Full control over project settings"},
		{ID: 6, Name: "manage_tasks", Description: "Create/edit/move/archive tasks"},
		{ID: 8, Name: "manage_tags", Description: "Manage project tags"},
	}

	for _, p := range permissions {
		_, err := pool.Exec(ctx, `
			INSERT INTO permission (id, name, description)
			VALUES ($1, $2, $3)
		`, p.ID, p.Name, p.Description)
		require.NoError(t, err, "failed to seed permission %d", p.ID)
	}

	return permissions
}

func TestPermissionRepository_GetAllPermissions_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewPermissionRepository(pool)
	ctx := context.Background()

	truncatePermissions(t, pool)
	expected := seedPermissions(t, pool)

	permissions, err := repo.GetAllPermissions(ctx)

	require.NoError(t, err)
	require.Len(t, permissions, len(expected))

	for i, p := range permissions {
		assert.Equal(t, expected[i].ID, p.ID)
		assert.Equal(t, expected[i].Name, p.Name)
		assert.Equal(t, expected[i].Description, p.Description)
	}
}

func TestPermissionRepository_GetAllPermissions_Empty(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewPermissionRepository(pool)
	ctx := context.Background()

	truncatePermissions(t, pool)

	permissions, err := repo.GetAllPermissions(ctx)

	require.NoError(t, err)
	assert.Empty(t, permissions)
}

func TestPermissionRepository_GetPermissionByID_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewPermissionRepository(pool)
	ctx := context.Background()

	truncatePermissions(t, pool)
	seedPermissions(t, pool)

	permission, err := repo.GetPermissionByID(ctx, 6)

	require.NoError(t, err)
	require.NotNil(t, permission)
	assert.Equal(t, 6, permission.ID)
	assert.Equal(t, "manage_tasks", permission.Name)
	assert.Equal(t, "Create/edit/move/archive tasks", permission.Description)
}

func TestPermissionRepository_GetPermissionByID_NotFound(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewPermissionRepository(pool)
	ctx := context.Background()

	truncatePermissions(t, pool)
	seedPermissions(t, pool)

	permission, err := repo.GetPermissionByID(ctx, 999)

	assert.ErrorIs(t, err, ErrPermissionNotFound)
	assert.Nil(t, permission)
}

func TestPermissionRepository_GetPermissionsByRoleID_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewPermissionRepository(pool)
	ctx := context.Background()

	truncatePermissions(t, pool)
	seedPermissions(t, pool)

	roleID := 10
	_, err := pool.Exec(ctx, `
		INSERT INTO role (id, project_uuid, name, description)
		VALUES ($1, NULL, 'Test Role', 'Role for testing')
	`, roleID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO role_permission (role_id, permission_id)
		VALUES ($1, $2), ($1, $3)
	`, roleID, 1, 6)
	require.NoError(t, err)

	permissions, err := repo.GetPermissionsByRoleID(ctx, roleID)

	require.NoError(t, err)
	require.Len(t, permissions, 2)

	assert.Equal(t, 1, permissions[0].ID)
	assert.Equal(t, "view_project", permissions[0].Name)
	assert.Equal(t, 6, permissions[1].ID)
	assert.Equal(t, "manage_tasks", permissions[1].Name)
}

func TestPermissionRepository_GetPermissionsByRoleID_Empty(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewPermissionRepository(pool)
	ctx := context.Background()

	truncatePermissions(t, pool)
	seedPermissions(t, pool)

	roleID := 11
	_, err := pool.Exec(ctx, `
		INSERT INTO role (id, project_uuid, name, description)
		VALUES ($1, NULL, 'Empty Role', 'Role with no permissions')
	`, roleID)
	require.NoError(t, err)

	permissions, err := repo.GetPermissionsByRoleID(ctx, roleID)

	require.NoError(t, err)
	assert.Empty(t, permissions)
}

func TestPermissionRepository_GetPermissionsByRoleID_NonExistentRole(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewPermissionRepository(pool)
	ctx := context.Background()

	truncatePermissions(t, pool)
	seedPermissions(t, pool)

	permissions, err := repo.GetPermissionsByRoleID(ctx, 9999)

	require.NoError(t, err)
	assert.Empty(t, permissions)
}

func TestPermissionRepository_GetPermissionByName_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewPermissionRepository(pool)
	ctx := context.Background()

	truncatePermissions(t, pool)
	seedPermissions(t, pool)

	permission, err := repo.GetPermissionByName(ctx, "manage_tags")

	require.NoError(t, err)
	require.NotNil(t, permission)
	assert.Equal(t, 8, permission.ID)
	assert.Equal(t, "manage_tags", permission.Name)
	assert.Equal(t, "Manage project tags", permission.Description)
}

func TestPermissionRepository_GetPermissionByName_NotFound(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewPermissionRepository(pool)
	ctx := context.Background()

	truncatePermissions(t, pool)
	seedPermissions(t, pool)

	permission, err := repo.GetPermissionByName(ctx, "non_existent_permission")

	assert.ErrorIs(t, err, ErrPermissionNotFound)
	assert.Nil(t, permission)
}

func TestPermissionRepository_FullWorkflow(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewPermissionRepository(pool)
	ctx := context.Background()

	truncatePermissions(t, pool)
	seedPermissions(t, pool)

	allPerms, err := repo.GetAllPermissions(ctx)
	require.NoError(t, err)
	assert.Len(t, allPerms, 4)

	permByID, err := repo.GetPermissionByID(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, "manage_project", permByID.Name)

	permByName, err := repo.GetPermissionByName(ctx, "view_project")
	require.NoError(t, err)
	assert.Equal(t, 1, permByName.ID)

	roleID := 20
	_, err = pool.Exec(ctx, `
		INSERT INTO role (id, project_uuid, name, description)
		VALUES ($1, NULL, 'Workflow Test Role', 'Testing full workflow')
	`, roleID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO role_permission (role_id, permission_id)
		VALUES ($1, $2), ($1, $3), ($1, $4)
	`, roleID, 1, 2, 6) // view_project, manage_project, manage_tasks
	require.NoError(t, err)

	rolePerms, err := repo.GetPermissionsByRoleID(ctx, roleID)
	require.NoError(t, err)
	require.Len(t, rolePerms, 3)

	permIDs := make(map[int]bool)
	for _, p := range rolePerms {
		permIDs[p.ID] = true
	}
	assert.True(t, permIDs[1], "role should have view_project")
	assert.True(t, permIDs[2], "role should have manage_project")
	assert.True(t, permIDs[6], "role should have manage_tasks")
	assert.False(t, permIDs[8], "role should NOT have manage_tags")
}
