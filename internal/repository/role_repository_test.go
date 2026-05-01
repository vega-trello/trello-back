//go:build integration
// +build integration

package repository

import (
	"context"
	"fmt"
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

	params := CreateRoleParams{
		Name:        fmt.Sprintf("Moderator_%s", uuid.New().String()[:8]),
		Description: stringPtr("Can edit tasks"),
	}

	role, err := repo.Create(ctx, projectUUID, owner, params)
	require.NoError(t, err)
	require.NotNil(t, role)

	assert.Greater(t, role.ID, 4)
	assert.Equal(t, params.Name, role.Name)
	assert.Equal(t, "Can edit tasks", *role.Description)
	assert.Equal(t, projectUUID, role.ProjectUUID)
}

func TestRoleRepository_Create_WithPermissions(t *testing.T) {
	repo, pool, owner := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	perm1 := createTestPermission(t, pool, "task.create", "Create tasks")
	perm2 := createTestPermission(t, pool, "task.update", "Update tasks")

	params := CreateRoleParams{
		Name:          fmt.Sprintf("Editor_%s", uuid.New().String()[:8]),
		Description:   stringPtr("Can edit"),
		PermissionIDs: []int{perm1, perm2},
	}

	role, err := repo.Create(ctx, projectUUID, owner, params)
	require.NoError(t, err)
	require.NotNil(t, role)
	assert.Greater(t, role.ID, 4)

	perms, err := repo.FindPermissions(ctx, projectUUID, role.ID, owner)
	require.NoError(t, err)
	require.Len(t, perms, 2)
}

func TestRoleRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, pool, owner := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	// Создаём роли с уникальными именами
	_, _ = repo.Create(ctx, projectUUID, owner, CreateRoleParams{Name: fmt.Sprintf("Role1_%s", uuid.New().String()[:8]), Description: stringPtr("Desc1")})
	_, _ = repo.Create(ctx, projectUUID, owner, CreateRoleParams{Name: fmt.Sprintf("Role2_%s", uuid.New().String()[:8]), Description: stringPtr("Desc2")})

	roles, err := repo.FindByProjectUUID(ctx, projectUUID, owner)
	require.NoError(t, err)
	// Системные роли имеют project_uuid = NULL, поэтому не попадут в выборку
	// Значит, должно быть ровно 2 созданные роли
	require.Len(t, roles, 2)
}

func TestRoleRepository_Update_Success(t *testing.T) {
	repo, pool, owner := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	// Создаём роль
	role, err := repo.Create(ctx, projectUUID, owner, CreateRoleParams{
		Name:        fmt.Sprintf("OldName_%s", uuid.New().String()[:8]),
		Description: stringPtr("OldDesc"),
	})
	require.NoError(t, err)
	require.NotNil(t, role)

	newName := "NewName"
	newDesc := "NewDesc"
	params := UpdateRoleParams{
		Name:        &newName,
		Description: &newDesc,
	}

	updated, err := repo.Update(ctx, projectUUID, role.ID, owner, params)
	require.NoError(t, err)
	require.NotNil(t, updated)

	assert.Equal(t, "NewName", updated.Name)
	assert.Equal(t, "NewDesc", *updated.Description)
}

func TestRoleRepository_Update_Permissions(t *testing.T) {
	repo, pool, owner := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	perm1 := createTestPermission(t, pool, "perm1", "Permission 1")
	perm2 := createTestPermission(t, pool, "perm2", "Permission 2")

	role, err := repo.Create(ctx, projectUUID, owner, CreateRoleParams{
		Name:          fmt.Sprintf("TestRole_%s", uuid.New().String()[:8]),
		PermissionIDs: []int{perm1},
	})
	require.NoError(t, err)
	require.NotNil(t, role)

	newPerms := []int{perm2}
	params := UpdateRoleParams{PermissionIDs: &newPerms}

	_, err = repo.Update(ctx, projectUUID, role.ID, owner, params)
	require.NoError(t, err)

	perms, _ := repo.FindPermissions(ctx, projectUUID, role.ID, owner)
	require.Len(t, perms, 1)
	assert.Equal(t, "perm2", perms[0].Name)
}

func TestRoleRepository_Delete_Success(t *testing.T) {
	repo, pool, owner := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	role, err := repo.Create(ctx, projectUUID, owner, CreateRoleParams{
		Name:        fmt.Sprintf("ToDelete_%s", uuid.New().String()[:8]),
		Description: stringPtr("Desc"),
	})
	require.NoError(t, err)
	require.NotNil(t, role)

	err = repo.Delete(ctx, projectUUID, role.ID, owner)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, projectUUID, role.ID, owner)
	assert.ErrorIs(t, err, ErrRoleNotFound)
}

func TestRoleRepository_AccessDenied(t *testing.T) {
	repo, pool, owner := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)

	outsider := createTestUser(t, pool, "outsider", "pass123")

	_, err := repo.Create(ctx, projectUUID, outsider, CreateRoleParams{Name: "Test", Description: stringPtr("Desc")})
	assert.ErrorIs(t, err, ErrAccessDenied)

	role, err := repo.Create(ctx, projectUUID, owner, CreateRoleParams{Name: fmt.Sprintf("Role1_%s", uuid.New().String()[:8]), Description: stringPtr("Desc")})
	require.NoError(t, err)
	_, err = repo.FindByID(ctx, projectUUID, role.ID, outsider)
	assert.ErrorIs(t, err, ErrAccessDenied)
}
