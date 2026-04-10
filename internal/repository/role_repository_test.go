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

func setupRoleRepo(t *testing.T) (*RoleRepository, *pgxpool.Pool) {
	t.Helper()
	pool := setupTestPool(t)
	repo := NewRoleRepository(pool)

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "TRUNCATE role_permission, role, permission RESTART IDENTITY CASCADE")
		cleanAllTables(t, pool)
	})

	return repo, pool
}

func createTestPermission(t *testing.T, pool *pgxpool.Pool, name, description string) int {
	t.Helper()
	ctx := context.Background()

	var permID int
	err := pool.QueryRow(ctx, `
		INSERT INTO permission (name, description)
		VALUES ($1, $2)
		RETURNING id
	`, name, description).Scan(&permID)
	require.NoError(t, err, "failed to create test permission")

	return permID
}

func createTestGlobalRole(t *testing.T, pool *pgxpool.Pool, name, description string) int {
	t.Helper()
	ctx := context.Background()

	var roleID int
	err := pool.QueryRow(ctx, `
		INSERT INTO role (project_uuid, name, description)
		VALUES (NULL, $1, $2)
		RETURNING id
	`, name, description).Scan(&roleID)
	require.NoError(t, err, "failed to create test global role")

	return roleID
}

func TestRoleRepository_Create_Success(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	input := RoleCreateInput{
		Name:        "Moderator",
		Description: "Can edit tasks",
	}

	role, err := repo.Create(ctx, projectUUID, input)
	require.NoError(t, err)
	require.NotNil(t, role)

	assert.Greater(t, role.ID, 0)
	assert.Equal(t, "Moderator", role.Name)
	assert.Equal(t, "Can edit tasks", role.Description)
	assert.Equal(t, projectUUID, role.ProjectUUID)
}

func TestRoleRepository_Create_WithPermissions(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	perm1 := createTestPermission(t, pool, "task.create", "Create tasks")
	perm2 := createTestPermission(t, pool, "task.update", "Update tasks")
	perm3 := createTestPermission(t, pool, "task.delete", "Delete tasks")

	input := RoleCreateInput{
		Name:          "Editor",
		Description:   "Can edit and delete",
		PermissionIDs: []int{perm1, perm2, perm3},
	}

	role, err := repo.Create(ctx, projectUUID, input)
	require.NoError(t, err)

	perms, err := repo.FindPermissions(ctx, projectUUID, role.ID)
	require.NoError(t, err)
	require.Len(t, perms, 3)
}

func TestRoleRepository_Create_DuplicateName(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	input := RoleCreateInput{
		Name:        "Duplicate",
		Description: "First",
	}

	_, err := repo.Create(ctx, projectUUID, input)
	require.NoError(t, err)

	input.Description = "Second"
	_, err = repo.Create(ctx, projectUUID, input)
	require.Error(t, err)
}

func TestRoleRepository_Create_SameNameDifferentProjects(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID1, _ := createTestProjectAndColumn(t, pool)
	projectUUID2, _ := createTestProjectAndColumn(t, pool)

	input := RoleCreateInput{
		Name:        "SameName",
		Description: "Project 1",
	}

	_, err := repo.Create(ctx, projectUUID1, input)
	require.NoError(t, err)

	input.Description = "Project 2"
	_, err = repo.Create(ctx, projectUUID2, input)
	require.NoError(t, err) // Разные проекты → можно
}

func TestRoleRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	createTestGlobalRole(t, pool, "GlobalRole1", "Global 1")
	createTestGlobalRole(t, pool, "GlobalRole2", "Global 2")

	_, _ = repo.Create(ctx, projectUUID, RoleCreateInput{Name: "ProjectRole1", Description: "Proj 1"})
	_, _ = repo.Create(ctx, projectUUID, RoleCreateInput{Name: "ProjectRole2", Description: "Proj 2"})

	roles, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, roles, 4)
}

func TestRoleRepository_FindByProjectUUID_OnlyGlobal(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	createTestGlobalRole(t, pool, "OnlyGlobal", "Global only")

	roles, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.True(t, roles[0].ProjectUUID == uuid.Nil)
}

func TestRoleRepository_FindByProjectUUID_Empty(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	roles, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	assert.Empty(t, roles)
}

func TestRoleRepository_Update_Success(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	role, _ := repo.Create(ctx, projectUUID, RoleCreateInput{
		Name:        "OldName",
		Description: "OldDesc",
	})

	newName := "NewName"
	newDesc := "NewDesc"
	updates := RoleUpdates{
		Name:        &newName,
		Description: &newDesc,
	}

	updated, err := repo.Update(ctx, projectUUID, role.ID, updates)
	require.NoError(t, err)

	assert.Equal(t, "NewName", updated.Name)
	assert.Equal(t, "NewDesc", updated.Description)
}

func TestRoleRepository_Update_Partial(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	role, _ := repo.Create(ctx, projectUUID, RoleCreateInput{
		Name:        "KeepName",
		Description: "KeepDesc",
	})

	newName := "NewName"
	updates := RoleUpdates{Name: &newName}

	updated, err := repo.Update(ctx, projectUUID, role.ID, updates)
	require.NoError(t, err)

	assert.Equal(t, "NewName", updated.Name)
	assert.Equal(t, "KeepDesc", updated.Description) // Не изменилось
}

func TestRoleRepository_Update_Permissions(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	perm1 := createTestPermission(t, pool, "perm1", "Permission 1")
	perm2 := createTestPermission(t, pool, "perm2", "Permission 2")
	perm3 := createTestPermission(t, pool, "perm3", "Permission 3")

	role, _ := repo.Create(ctx, projectUUID, RoleCreateInput{
		Name:          "TestRole",
		PermissionIDs: []int{perm1},
	})

	// Обновляем разрешения (полная замена)
	newPerms := []int{perm2, perm3}
	updates := RoleUpdates{PermissionIDs: &newPerms}

	_, err := repo.Update(ctx, projectUUID, role.ID, updates)
	require.NoError(t, err)

	// Проверяем, что старые разрешения удалены, новые добавлены
	perms, _ := repo.FindPermissions(ctx, projectUUID, role.ID)
	require.Len(t, perms, 2)
}

func TestRoleRepository_Update_NotFound(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	name := "test"
	updates := RoleUpdates{Name: &name}

	_, err := repo.Update(ctx, projectUUID, 99999, updates)
	require.Error(t, err)
}

func TestRoleRepository_Update_WrongProject(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()

	projectUUID1, _ := createTestProjectAndColumn(t, pool)
	projectUUID2, _ := createTestProjectAndColumn(t, pool)

	role, _ := repo.Create(ctx, projectUUID1, RoleCreateInput{Name: "Role1", Description: "Desc"})

	newName := "Changed"
	updates := RoleUpdates{Name: &newName}

	// Пытаемся обновить роль из проекта 1 через проект 2
	_, err := repo.Update(ctx, projectUUID2, role.ID, updates)
	require.Error(t, err)
}

func TestRoleRepository_Delete_Success(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	for i := 0; i < 4; i++ {
		_, _ = repo.Create(ctx, projectUUID, RoleCreateInput{
			Name:        fmt.Sprintf("Dummy%d", i),
			Description: "Dummy role for ID consumption",
		})
	}

	role, err := repo.Create(ctx, projectUUID, RoleCreateInput{
		Name:        "ToDelete",
		Description: "Delete",
	})
	require.NoError(t, err)
	require.Greater(t, role.ID, 4) // Убеждаемся, что роль не системная

	// Теперь удаление должно пройти успешно
	err = repo.Delete(ctx, projectUUID, role.ID)
	assert.NoError(t, err)

	// Проверяем, что роль действительно удалена
	_, err = repo.findByID(ctx, projectUUID, role.ID)
	assert.Error(t, err) // Должна быть ошибка "not found"
}

func TestRoleRepository_Delete_SystemRole(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	// Пытаемся удалить системную роль (ID <= 4)
	err := repo.Delete(ctx, projectUUID, 1) // RoleOwner
	assert.ErrorIs(t, err, ErrCannotDeleteSystemRole)

	err = repo.Delete(ctx, projectUUID, 2) // RoleAdmin
	assert.ErrorIs(t, err, ErrCannotDeleteSystemRole)

	err = repo.Delete(ctx, projectUUID, 3) // RoleMember
	assert.ErrorIs(t, err, ErrCannotDeleteSystemRole)

	err = repo.Delete(ctx, projectUUID, 4) // RoleViewer
	assert.ErrorIs(t, err, ErrCannotDeleteSystemRole)
}

func TestRoleRepository_Delete_NotFound(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	err := repo.Delete(ctx, projectUUID, 99999)
	assert.Error(t, err)
}

func TestRoleRepository_Delete_WrongProject(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()

	projectUUID1, _ := createTestProjectAndColumn(t, pool)
	projectUUID2, _ := createTestProjectAndColumn(t, pool)

	role, _ := repo.Create(ctx, projectUUID1, RoleCreateInput{Name: "WrongProj", Description: "Desc"})

	// Пытаемся удалить через проект 2
	err := repo.Delete(ctx, projectUUID2, role.ID)
	assert.Error(t, err)
}

func TestRoleRepository_FindPermissions_Success(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	perm1 := createTestPermission(t, pool, "perm1", "Permission 1")
	perm2 := createTestPermission(t, pool, "perm2", "Permission 2")
	perm3 := createTestPermission(t, pool, "perm3", "Permission 3")

	role, _ := repo.Create(ctx, projectUUID, RoleCreateInput{
		Name:          "TestRole",
		PermissionIDs: []int{perm1, perm2, perm3},
	})

	perms, err := repo.FindPermissions(ctx, projectUUID, role.ID)
	require.NoError(t, err)
	require.Len(t, perms, 3)

	// Проверяем сортировку по имени
	assert.Equal(t, "perm1", perms[0].Name)
	assert.Equal(t, "perm2", perms[1].Name)
	assert.Equal(t, "perm3", perms[2].Name)
}

func TestRoleRepository_FindPermissions_Empty(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	role, _ := repo.Create(ctx, projectUUID, RoleCreateInput{Name: "NoPerms", Description: "No permissions"})

	perms, err := repo.FindPermissions(ctx, projectUUID, role.ID)
	require.NoError(t, err)
	assert.Empty(t, perms)
}

func TestRoleRepository_FindPermissions_WrongProject(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()

	projectUUID1, _ := createTestProjectAndColumn(t, pool)
	projectUUID2, _ := createTestProjectAndColumn(t, pool)

	role, _ := repo.Create(ctx, projectUUID1, RoleCreateInput{Name: "Role1", Description: "Desc"})

	// Пытаемся получить разрешения через проект 2
	perms, err := repo.FindPermissions(ctx, projectUUID2, role.ID)
	require.Error(t, err)
	assert.Empty(t, perms)
}

func TestRoleRepository_AssignPermissions_Success(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	perm1 := createTestPermission(t, pool, "perm1", "Permission 1")
	perm2 := createTestPermission(t, pool, "perm2", "Permission 2")

	role, _ := repo.Create(ctx, projectUUID, RoleCreateInput{
		Name:          "TestRole",
		PermissionIDs: []int{perm1},
	})

	// Назначаем новые разрешения (полная замена)
	newPerms := []int{perm2}
	err := repo.AssignPermissions(ctx, projectUUID, role.ID, newPerms)
	require.NoError(t, err)

	// Проверяем, что perm1 удалён, perm2 добавлен
	perms, _ := repo.FindPermissions(ctx, projectUUID, role.ID)
	require.Len(t, perms, 1)
	assert.Equal(t, "perm2", perms[0].Name)
}

func TestRoleRepository_AssignPermissions_ClearAll(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	perm1 := createTestPermission(t, pool, "perm1", "Permission 1")

	role, _ := repo.Create(ctx, projectUUID, RoleCreateInput{
		Name:          "TestRole",
		PermissionIDs: []int{perm1},
	})

	// Очищаем все разрешения
	err := repo.AssignPermissions(ctx, projectUUID, role.ID, []int{})
	require.NoError(t, err)

	perms, _ := repo.FindPermissions(ctx, projectUUID, role.ID)
	assert.Empty(t, perms)
}

func TestRoleRepository_AssignPermissions_InvalidPermission(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	role, _ := repo.Create(ctx, projectUUID, RoleCreateInput{Name: "TestRole", Description: "Desc"})

	// Пытаемся назначить несуществующее разрешение
	err := repo.AssignPermissions(ctx, projectUUID, role.ID, []int{99999})
	require.Error(t, err)
}

func TestRoleRepository_HasPermission_Success(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	permID := createTestPermission(t, pool, "task.create", "Create tasks")

	role, _ := repo.Create(ctx, projectUUID, RoleCreateInput{
		Name:          "TestRole",
		PermissionIDs: []int{permID},
	})

	has, err := repo.HasPermission(ctx, role.ID, "task.create")
	require.NoError(t, err)
	assert.True(t, has)
}

func TestRoleRepository_HasPermission_NotAssigned(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	createTestPermission(t, pool, "task.create", "Create tasks")

	role, _ := repo.Create(ctx, projectUUID, RoleCreateInput{
		Name: "TestRole",
		// Без разрешений
	})

	has, err := repo.HasPermission(ctx, role.ID, "task.create")
	require.NoError(t, err)
	assert.False(t, has)
}

func TestRoleRepository_HasPermission_NonExistent(t *testing.T) {
	repo, pool := setupRoleRepo(t)
	ctx := context.Background()
	projectUUID, _ := createTestProjectAndColumn(t, pool)

	role, _ := repo.Create(ctx, projectUUID, RoleCreateInput{Name: "TestRole", Description: "Desc"})

	has, err := repo.HasPermission(ctx, role.ID, "nonexistent.permission")
	require.NoError(t, err)
	assert.False(t, has)
}
