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

func setupRoleRepo(t *testing.T) (*RoleRepository, *pgxpool.Pool, uuid.UUID) {
	t.Helper()
	pool := setupTestPool(t)
	owner := createTestUser(t, pool, "role_owner", "pass123")
	repo := NewRoleRepository(pool)
	return repo, pool, owner
}

func createTestPermission(t *testing.T, pool *pgxpool.Pool, name, description string) int {
	t.Helper()
	ctx := context.Background()
	var permID int
	err := pool.QueryRow(ctx, `INSERT INTO permission (name, description) VALUES ($1, $2) RETURNING id`, name, description).Scan(&permID)
	require.NoError(t, err)
	return permID
}

func TestRoleRepository_Create_Success(t *testing.T) {
	repo, pool, owner := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	perm1 := createTestPermission(t, pool, "task.create", "Create tasks")
	perm2 := createTestPermission(t, pool, "task.update", "Update tasks")

	role, err := repo.Create(ctx, projectUUID, owner, "Moderator", stringPtr("Can edit tasks"), []int{perm1, perm2})
	require.NoError(t, err)
	require.NotNil(t, role)

	assert.Equal(t, "Moderator", role.Name)
	assert.Equal(t, projectUUID, *role.ProjectUUID)
	assert.Greater(t, role.ID, 4) // Системные роли имеют ID 1-4
}

func TestRoleRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, pool, owner := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	perm := createTestPermission(t, pool, "test.perm", "Test")
	_, _ = repo.Create(ctx, projectUUID, owner, "Role1", stringPtr("Desc1"), []int{perm})
	_, _ = repo.Create(ctx, projectUUID, owner, "Role2", stringPtr("Desc2"), []int{perm})

	roles, err := repo.FindByProjectUUID(ctx, projectUUID, owner)
	require.NoError(t, err)
	require.Len(t, roles, 2)
	assert.Equal(t, "Role1", roles[0].Name)
	assert.Equal(t, "Role2", roles[1].Name)
}

func TestRoleRepository_FindByID_Success(t *testing.T) {
	repo, pool, owner := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	perm := createTestPermission(t, pool, "find.perm", "Find")
	created, _ := repo.Create(ctx, projectUUID, owner, "FindMe", stringPtr("Desc"), []int{perm})

	role, err := repo.FindByID(ctx, projectUUID, created.ID, owner)
	require.NoError(t, err)
	assert.Equal(t, created.ID, role.ID)
	assert.Equal(t, "FindMe", role.Name)
}

func TestRoleRepository_Update_Success(t *testing.T) {
	repo, pool, owner := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	perm := createTestPermission(t, pool, "update.perm", "Update")
	role, _ := repo.Create(ctx, projectUUID, owner, "OldName", stringPtr("OldDesc"), []int{perm})

	newName := "NewName"
	newDesc := "Updated description"
	updated, err := repo.Update(ctx, projectUUID, role.ID, owner, newName, stringPtr(newDesc), []int{perm})
	require.NoError(t, err)

	assert.Equal(t, "NewName", updated.Name)
	assert.Equal(t, "Updated description", *updated.Description)
}

func TestRoleRepository_Delete_Success(t *testing.T) {
	repo, pool, owner := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	perm := createTestPermission(t, pool, "delete.perm", "Delete")
	role, _ := repo.Create(ctx, projectUUID, owner, "ToDelete", stringPtr("Desc"), []int{perm})

	err := repo.Delete(ctx, projectUUID, role.ID, owner)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, projectUUID, role.ID, owner)
	assert.ErrorIs(t, err, ErrRoleNotFound)
}

func TestRoleRepository_CannotDeleteSystemRole(t *testing.T) {
	repo, pool, owner := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	// Системная роль имеет ID=1 и project_uuid=NULL
	err := repo.Delete(ctx, projectUUID, 1, owner)
	assert.ErrorIs(t, err, ErrCannotDeleteSystemRole)
}

func TestRoleRepository_FindPermissions_Success(t *testing.T) {
	repo, pool, owner := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	perm1 := createTestPermission(t, pool, "perm1", "Permission 1")
	perm2 := createTestPermission(t, pool, "perm2", "Permission 2")
	role, _ := repo.Create(ctx, projectUUID, owner, "TestRole", stringPtr("Desc"), []int{perm1, perm2})

	perms, err := repo.FindPermissions(ctx, projectUUID, role.ID, owner)
	require.NoError(t, err)
	require.Len(t, perms, 2)
	assert.Contains(t, []string{perms[0].Name, perms[1].Name}, "perm1")
}
