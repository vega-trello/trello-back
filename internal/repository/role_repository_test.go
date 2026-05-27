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

func TestRoleRepository_Create_Success(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "role_owner", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	permID1 := ensurePermission(t, pool, "manage_tasks", "Manage tasks")
	permID2 := ensurePermission(t, pool, "view_project", "View project")

	description := "Test role description"
	role, err := repo.Create(ctx, projectUUID, ownerUUID, "Test Role", &description, []int{permID1, permID2})

	require.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "Test Role", role.Name)
	assert.Equal(t, projectUUID, *role.ProjectUUID)
	assert.Equal(t, description, *role.Description)

	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM role_permission WHERE role_id = $1`, role.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestRoleRepository_Create_NoAccess(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "noaccess_owner", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	otherUserUUID := createTestUser(t, pool, "other_user", "pass123")

	_, err := repo.Create(ctx, projectUUID, otherUserUUID, "Test Role", nil, []int{})

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestRoleRepository_FindByProjectUUID_Success(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "find_owner", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	_, err := repo.Create(ctx, projectUUID, ownerUUID, "Custom Role", nil, []int{})
	require.NoError(t, err)

	roles, err := repo.FindByProjectUUID(ctx, projectUUID, ownerUUID)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(roles), 1)

	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
	}
	assert.Contains(t, roleNames, "Custom Role")
}

func TestRoleRepository_FindByID_Success(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "findid_owner", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	role, err := repo.Create(ctx, projectUUID, ownerUUID, "FindMe Role", nil, []int{})
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, projectUUID, role.ID, ownerUUID)

	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "FindMe Role", found.Name)
	assert.Equal(t, role.ID, found.ID)
}

func TestRoleRepository_FindByID_NotFound(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "notfound_owner", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	_, err := repo.FindByID(ctx, projectUUID, 99999, ownerUUID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrRoleNotFound)
}

func TestRoleRepository_Update_Success(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "update_owner", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	permID1 := ensurePermission(t, pool, "manage_tasks", "Manage tasks")
	role, err := repo.Create(ctx, projectUUID, ownerUUID, "Old Name", nil, []int{permID1})
	require.NoError(t, err)

	permID2 := ensurePermission(t, pool, "view_project", "View project")
	newDesc := "Updated description"
	updated, err := repo.Update(ctx, projectUUID, role.ID, ownerUUID, "New Name", &newDesc, []int{permID2})

	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, newDesc, *updated.Description)

	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM role_permission WHERE role_id = $1`, role.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRoleRepository_Update_SystemRole(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "sysupdate_owner", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	_, err := repo.Update(ctx, projectUUID, RoleOwner, ownerUUID, "Hacked Name", nil, []int{})

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrCannotDeleteSystemRole)
}

func TestRoleRepository_Delete_Success(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "delete_owner", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	role, err := repo.Create(ctx, projectUUID, ownerUUID, "Temporary Role", nil, []int{})
	require.NoError(t, err)

	err = repo.Delete(ctx, projectUUID, role.ID, ownerUUID)

	assert.NoError(t, err)

	var exists bool
	err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM role WHERE id = $1)`, role.ID).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRoleRepository_Delete_InUse(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "inuse_owner", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	err := repo.Delete(ctx, projectUUID, RoleOwner, ownerUUID)

	assert.Error(t, err)
	assert.True(t, err == ErrRoleInUse || err == ErrCannotDeleteSystemRole)
}

func TestRoleRepository_FindPermissions_Success(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "perms_owner", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	permID1 := ensurePermission(t, pool, "manage_tasks", "Manage tasks")
	permID2 := ensurePermission(t, pool, "view_project", "View project")
	role, err := repo.Create(ctx, projectUUID, ownerUUID, "PermTest Role", nil, []int{permID1, permID2})
	require.NoError(t, err)

	perms, err := repo.FindPermissions(ctx, projectUUID, role.ID, ownerUUID)

	require.NoError(t, err)
	assert.Len(t, perms, 2)
	permNames := []string{perms[0].Name, perms[1].Name}
	assert.Contains(t, permNames, "manage_tasks")
	assert.Contains(t, permNames, "view_project")
}

func TestRoleRepository_GetUserRole_Success(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "getrole_user", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	role, err := repo.GetUserRole(ctx, projectUUID, ownerUUID)

	require.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "Owner", role.Name)
	assert.Equal(t, RoleOwner, role.ID)
}

func TestRoleRepository_GetUserRole_CustomRole(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "customrole_owner", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	userUUID := createTestUser(t, pool, "customrole_user", "pass123")
	_, err := pool.Exec(ctx, `
		INSERT INTO project_member (project_uuid, user_uuid, role_id, joined_at)
		VALUES ($1, $2, 3, NOW())
	`, projectUUID, userUUID)
	require.NoError(t, err)

	roleID := createCustomRoleForMember(t, pool, projectUUID, "Custom", []int{})

	_, err = pool.Exec(ctx, `
		UPDATE project_member SET role_id = $1 
		WHERE project_uuid = $2 AND user_uuid = $3
	`, roleID, projectUUID, userUUID)
	require.NoError(t, err)

	role, err := repo.GetUserRole(ctx, projectUUID, userUUID)

	require.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "Custom", role.Name)
	assert.Equal(t, projectUUID, *role.ProjectUUID)
}

func TestRoleRepository_GetUserRole_NotFound(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	otherUserUUID := createTestUser(t, pool, "norole_user", "pass123")
	projectUUID := uuid.New()

	_, err := repo.GetUserRole(ctx, projectUUID, otherUserUUID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrRoleNotFound)
}

func TestRoleRepository_GetPermissionsByRoleID_Success(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "getperms_owner", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	permID1 := ensurePermission(t, pool, "manage_tasks", "Manage tasks")
	permID2 := ensurePermission(t, pool, "view_project", "View project")
	role, err := repo.Create(ctx, projectUUID, ownerUUID, "GetPerms Role", nil, []int{permID1, permID2})
	require.NoError(t, err)

	perms, err := repo.GetPermissionsByRoleID(ctx, role.ID)

	require.NoError(t, err)
	assert.Len(t, perms, 2)
	permNames := []string{perms[0].Name, perms[1].Name}
	assert.Contains(t, permNames, "manage_tasks")
	assert.Contains(t, permNames, "view_project")
}

func TestRoleRepository_GetPermissionsByRoleID_NoPermissions(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	ownerUUID := createTestUser(t, pool, "emptyperms_owner", "pass123")
	projectUUID := createTestProject(t, pool, ownerUUID)

	role, err := repo.Create(ctx, projectUUID, ownerUUID, "Empty Role", nil, []int{})
	require.NoError(t, err)

	perms, err := repo.GetPermissionsByRoleID(ctx, role.ID)

	require.NoError(t, err)
	assert.Empty(t, perms)
}

func TestRoleRepository_GetPermissionsByRoleID_SystemRole(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	_, err := repo.GetPermissionsByRoleID(ctx, RoleOwner)

	assert.NoError(t, err)
}

func TestRoleRepository_GetPermissionsByRoleID_NotFound(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	perms, err := repo.GetPermissionsByRoleID(ctx, 99999)

	assert.NoError(t, err)
	assert.Empty(t, perms)
}

func TestRoleRepository_GetPermissionNamesByID_Success(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	permID1 := ensurePermission(t, pool, "manage_tasks", "Manage tasks")
	permID2 := ensurePermission(t, pool, "view_project", "View project")
	permID3 := ensurePermission(t, pool, "manage_roles", "Manage roles")

	result, err := repo.GetPermissionNamesByID(ctx, []int{permID1, permID2, permID3})

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "manage_tasks", result[permID1])
	assert.Equal(t, "view_project", result[permID2])
	assert.Equal(t, "manage_roles", result[permID3])
}

func TestRoleRepository_GetPermissionNamesByID_EmptySlice(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	result, err := repo.GetPermissionNamesByID(ctx, []int{})

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestRoleRepository_GetPermissionNamesByID_NotFound(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	_, err := repo.GetPermissionNamesByID(ctx, []int{99999})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission ID 99999 not found")
}

func TestRoleRepository_GetPermissionNamesByID_Mixed(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	permID := ensurePermission(t, pool, "manage_tasks", "Manage tasks")

	_, err := repo.GetPermissionNamesByID(ctx, []int{permID, 99999})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission ID 99999 not found")
}

func TestRoleRepository_GetPermissionNamesByID_DuplicateIDs(t *testing.T) {
	pool := setupTestPool(t)
	ctx := context.Background()
	repo := NewRoleRepository(pool)

	permID := ensurePermission(t, pool, "manage_tasks", "Manage tasks")

	result, err := repo.GetPermissionNamesByID(ctx, []int{permID, permID, permID})

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "manage_tasks", result[permID])
}

func ensurePermission(t *testing.T, pool *pgxpool.Pool, name, description string) int {
	t.Helper()
	ctx := context.Background()

	var permID int
	err := pool.QueryRow(ctx, `
		INSERT INTO permission (name, description)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET description = $2
		RETURNING id
	`, name, description).Scan(&permID)
	require.NoError(t, err)
	return permID
}

func createCustomRoleForMember(t *testing.T, pool *pgxpool.Pool, projectUUID uuid.UUID, name string, permIDs []int) int {
	t.Helper()
	ctx := context.Background()

	var roleID int
	err := pool.QueryRow(ctx, `
		INSERT INTO role (project_uuid, name, description)
		VALUES ($1, $2, $3)
		RETURNING id
	`, projectUUID, name, "Custom role").Scan(&roleID)
	require.NoError(t, err)

	for _, permID := range permIDs {
		_, err := pool.Exec(ctx, `
			INSERT INTO role_permission (role_id, permission_id)
			VALUES ($1, $2)
		`, roleID, permID)
		require.NoError(t, err)
	}
	return roleID
}
