//go:build integration
// +build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMemberRepo(t *testing.T) (*MemberRepository, *pgxpool.Pool, uuid.UUID) {
	t.Helper()
	pool := setupTestPool(t)
	owner := createTestUser(t, pool, "member_owner", "pass123")
	repo := NewMemberRepository(pool)
	return repo, pool, owner
}

func TestMemberRepository_Create_Success(t *testing.T) {
	repo, pool, owner := setupMemberRepo(t)
	ctx := context.Background()

	projectUUID := createTestProject(t, pool, owner)
	userUUID := createTestUser(t, pool, "new_member", "pass123")

	member, err := repo.Create(ctx, projectUUID, userUUID, RoleMember)
	require.NoError(t, err)
	require.NotNil(t, member)

	assert.Equal(t, projectUUID, member.ProjectUUID)
	assert.Equal(t, userUUID, member.UserUUID)
	assert.Equal(t, RoleMember, member.RoleID)
	assert.WithinDuration(t, time.Now(), member.JoinedAt, 2*time.Second)
}

func TestMemberRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, pool, owner := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	u1 := createTestUser(t, pool, "find_u1", "pass123")
	u2 := createTestUser(t, pool, "find_u2", "pass123")

	_, err := repo.Create(ctx, projectUUID, u1, RoleAdmin)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID, u2, RoleMember)
	require.NoError(t, err)

	members, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, members, 3)
}

func TestMemberRepository_FindByProjectUUID_Empty_Success(t *testing.T) {
	repo, pool, _ := setupMemberRepo(t) // 🔹 _ вместо owner
	ctx := context.Background()

	// Создаём проект без участников
	projectUUID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO project (uuid, title, description, created_at, updated_at)
		VALUES ($1, 'Empty Project', '', NOW(), NOW())
	`, projectUUID)
	require.NoError(t, err)

	members, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)

	assert.Empty(t, members)
	assert.NotNil(t, members)
}

func TestMemberRepository_FindByProjectAndUser_Success(t *testing.T) {
	repo, pool, owner := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	userUUID := createTestUser(t, pool, "find_single", "pass123")

	_, err := repo.Create(ctx, projectUUID, userUUID, RoleAdmin)
	require.NoError(t, err)

	member, err := repo.FindByProjectAndUser(ctx, projectUUID, userUUID)
	require.NoError(t, err)
	assert.Equal(t, RoleAdmin, member.RoleID)
}

func TestMemberRepository_FindByProjectAndUser_NotFound(t *testing.T) {
	repo, pool, owner := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	unknownUUID := uuid.New()

	_, err := repo.FindByProjectAndUser(ctx, projectUUID, unknownUUID)
	assert.ErrorIs(t, err, ErrMemberNotFound)
}

func TestMemberRepository_Update_Promote(t *testing.T) {
	repo, pool, owner := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	userUUID := createTestUser(t, pool, "promote_user", "pass123")

	_, err := repo.Create(ctx, projectUUID, userUUID, RoleMember)
	require.NoError(t, err)

	member, err := repo.Update(ctx, projectUUID, userUUID, RoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, RoleAdmin, member.RoleID)
}

func TestMemberRepository_Update_NotFound(t *testing.T) {
	repo, pool, owner := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	unknownUUID := uuid.New()

	_, err := repo.Update(ctx, projectUUID, unknownUUID, RoleAdmin)
	assert.ErrorIs(t, err, ErrMemberNotFound)
}

func TestMemberRepository_Delete_Success(t *testing.T) {
	repo, pool, owner := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	userUUID := createTestUser(t, pool, "delete_user", "pass123")

	created, err := repo.Create(ctx, projectUUID, userUUID, RoleMember)
	require.NoError(t, err)

	err = repo.Delete(ctx, created.ProjectUUID, created.UserUUID)
	assert.NoError(t, err)

	_, err = repo.FindByProjectAndUser(ctx, projectUUID, userUUID)
	assert.ErrorIs(t, err, ErrMemberNotFound)
}

func TestMemberRepository_Delete_NotFound(t *testing.T) {
	repo, pool, owner := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool, owner)
	unknownUUID := uuid.New()

	err := repo.Delete(ctx, projectUUID, unknownUUID)
	assert.ErrorIs(t, err, ErrMemberNotFound)
}

func TestMemberRepository_FindByProjectUUIDWithDetails_Empty_Success(t *testing.T) {
	repo, pool, _ := setupMemberRepo(t)
	ctx := context.Background()

	projectUUID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO project (uuid, title, description, created_at, updated_at)
		VALUES ($1, 'Empty Details Project', '', NOW(), NOW())
	`, projectUUID)
	require.NoError(t, err)

	members, err := repo.FindByProjectUUIDWithDetails(ctx, projectUUID)
	require.NoError(t, err)

	assert.Empty(t, members)
	assert.NotNil(t, members)
}
