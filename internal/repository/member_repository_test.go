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
	"github.com/vega-trello/trello-back/internal/model"
)

// setupMemberRepo инициализирует репозиторий и подключение к тестовой БД
func setupMemberRepo(t *testing.T) (*MemberRepository, *pgxpool.Pool) {
	t.Helper()
	pool := setupTestPool(t)
	
	ensureGlobalRoles(t, pool)

	repo := NewMemberRepository(pool)
	t.Cleanup(func() { cleanAllTables(t, pool) })
	return repo, pool
}

func TestMemberRepository_Create_Success(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()

	projectUUID := createTestProject(t, pool)
	userUUID := createTestBaseUser(t, pool)

	member, err := repo.Create(ctx, projectUUID, userUUID, model.RoleMember)
	require.NoError(t, err)
	require.NotNil(t, member)

	assert.Equal(t, projectUUID, member.ProjectUUID)
	assert.Equal(t, userUUID, member.UserUUID)
	assert.Equal(t, model.RoleMember, member.RoleID)
	assert.WithinDuration(t, time.Now(), member.JoinedAt, 2*time.Second)
}

func TestMemberRepository_Create_DifferentRoles(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	userUUID := createTestBaseUser(t, pool)

	member, err := repo.Create(ctx, projectUUID, userUUID, model.RoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, model.RoleAdmin, member.RoleID)
}

func TestMemberRepository_Create_DuplicateMember(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	userUUID := createTestBaseUser(t, pool)

	_, err := repo.Create(ctx, projectUUID, userUUID, model.RoleMember)
	require.NoError(t, err)

	_, err = repo.Create(ctx, projectUUID, userUUID, model.RoleAdmin)
	assert.ErrorIs(t, err, ErrMemberAlreadyExists)
}

func TestMemberRepository_Create_SameUserDifferentProjects(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID1 := createTestProject(t, pool)
	projectUUID2 := createTestProject(t, pool)
	userUUID := createTestBaseUser(t, pool)

	m1, err := repo.Create(ctx, projectUUID1, userUUID, model.RoleMember)
	require.NoError(t, err)

	m2, err := repo.Create(ctx, projectUUID2, userUUID, model.RoleAdmin)
	require.NoError(t, err)

	assert.Equal(t, projectUUID1, m1.ProjectUUID)
	assert.Equal(t, projectUUID2, m2.ProjectUUID)
}

func TestMemberRepository_Create_MultipleUsersInOneProject(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	u1 := createTestBaseUser(t, pool)
	u2 := createTestBaseUser(t, pool)
	u3 := createTestBaseUser(t, pool)

	_, err := repo.Create(ctx, projectUUID, u1, model.RoleAdmin)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID, u2, model.RoleMember)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID, u3, model.RoleViewer)
	require.NoError(t, err)
}

func TestMemberRepository_FindByProjectUUID_Success(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	u1 := createTestBaseUser(t, pool)
	u2 := createTestBaseUser(t, pool)

	_, err := repo.Create(ctx, projectUUID, u1, model.RoleAdmin)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID, u2, model.RoleMember)
	require.NoError(t, err)

	members, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, members, 2)
}

func TestMemberRepository_FindByProjectUUID_Empty(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)

	members, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	assert.Empty(t, members)
}

func TestMemberRepository_FindByProjectUUID_SortedByJoinedAt(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	u1 := createTestBaseUser(t, pool)
	u2 := createTestBaseUser(t, pool)
	u3 := createTestBaseUser(t, pool)

	_, _ = repo.Create(ctx, projectUUID, u1, model.RoleViewer)
	time.Sleep(10 * time.Millisecond)
	_, _ = repo.Create(ctx, projectUUID, u2, model.RoleMember)
	time.Sleep(10 * time.Millisecond)
	_, _ = repo.Create(ctx, projectUUID, u3, model.RoleAdmin)

	members, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, members, 3)

	assert.True(t, members[0].JoinedAt.Before(members[1].JoinedAt))
	assert.True(t, members[1].JoinedAt.Before(members[2].JoinedAt))
}

func TestMemberRepository_FindByProjectUUID_Isolation(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	p1 := createTestProject(t, pool)
	p2 := createTestProject(t, pool)
	u := createTestBaseUser(t, pool)

	_, err := repo.Create(ctx, p1, u, model.RoleMember)
	require.NoError(t, err)

	members, err := repo.FindByProjectUUID(ctx, p1)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, p1, members[0].ProjectUUID)

	members2, err := repo.FindByProjectUUID(ctx, p2)
	require.NoError(t, err)
	assert.Empty(t, members2)
}

func TestMemberRepository_FindByProjectAndUser_Success(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	userUUID := createTestBaseUser(t, pool)

	_, err := repo.Create(ctx, projectUUID, userUUID, model.RoleAdmin)
	require.NoError(t, err)

	member, err := repo.FindByProjectAndUser(ctx, projectUUID, userUUID)
	require.NoError(t, err)
	assert.Equal(t, model.RoleAdmin, member.RoleID)
}

func TestMemberRepository_FindByProjectAndUser_NotFound(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	wrongUserUUID := uuid.New()

	_, err := repo.FindByProjectAndUser(ctx, projectUUID, wrongUserUUID)
	assert.ErrorIs(t, err, ErrMemberNotFound)
}

func TestMemberRepository_Update_Promote(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	userUUID := createTestBaseUser(t, pool)

	_, err := repo.Create(ctx, projectUUID, userUUID, model.RoleMember)
	require.NoError(t, err)

	member, err := repo.Update(ctx, projectUUID, userUUID, model.RoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, model.RoleAdmin, member.RoleID)
}

func TestMemberRepository_Update_Demote(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	userUUID := createTestBaseUser(t, pool)

	_, err := repo.Create(ctx, projectUUID, userUUID, model.RoleAdmin)
	require.NoError(t, err)

	member, err := repo.Update(ctx, projectUUID, userUUID, model.RoleViewer)
	require.NoError(t, err)
	assert.Equal(t, model.RoleViewer, member.RoleID)
}

func TestMemberRepository_Update_NotFound(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	userUUID := createTestBaseUser(t, pool)

	_, err := repo.Update(ctx, projectUUID, userUUID, model.RoleAdmin)
	assert.ErrorIs(t, err, ErrMemberNotFound)
}

func TestMemberRepository_Delete_Success(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	userUUID := createTestBaseUser(t, pool)

	created, err := repo.Create(ctx, projectUUID, userUUID, model.RoleMember)
	require.NoError(t, err)

	err = repo.Delete(ctx, created.ProjectUUID, created.UserUUID)
	assert.NoError(t, err)

	_, err = repo.FindByProjectAndUser(ctx, projectUUID, userUUID)
	assert.ErrorIs(t, err, ErrMemberNotFound)
}

func TestMemberRepository_Delete_NotFound(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	userUUID := createTestBaseUser(t, pool)

	err := repo.Delete(ctx, projectUUID, userUUID)
	assert.ErrorIs(t, err, ErrMemberNotFound)
}

func TestMemberRepository_Delete_LeavesOthersIntact(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	u1 := createTestBaseUser(t, pool)
	u2 := createTestBaseUser(t, pool)
	u3 := createTestBaseUser(t, pool)

	_, err := repo.Create(ctx, projectUUID, u1, model.RoleAdmin)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID, u2, model.RoleMember)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID, u3, model.RoleViewer)
	require.NoError(t, err)

	err = repo.Delete(ctx, projectUUID, u2)
	require.NoError(t, err)

	members, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, members, 2)

	for _, m := range members {
		assert.NotEqual(t, u2, m.UserUUID)
	}
}

func TestMemberRepository_FullWorkflow(t *testing.T) {
	repo, pool := setupMemberRepo(t)
	ctx := context.Background()
	projectUUID := createTestProject(t, pool)
	adminUUID := createTestBaseUser(t, pool)
	memberUUID := createTestBaseUser(t, pool)
	viewerUUID := createTestBaseUser(t, pool)

	_, err := repo.Create(ctx, projectUUID, adminUUID, model.RoleAdmin)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID, memberUUID, model.RoleMember)
	require.NoError(t, err)
	_, err = repo.Create(ctx, projectUUID, viewerUUID, model.RoleViewer)
	require.NoError(t, err)

	members, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, members, 3)

	admin, err := repo.FindByProjectAndUser(ctx, projectUUID, adminUUID)
	require.NoError(t, err)
	assert.Equal(t, model.RoleAdmin, admin.RoleID)

	updated, err := repo.Update(ctx, projectUUID, viewerUUID, model.RoleMember)
	require.NoError(t, err)
	assert.Equal(t, model.RoleMember, updated.RoleID)

	err = repo.Delete(ctx, projectUUID, memberUUID)
	require.NoError(t, err)

	finalMembers, err := repo.FindByProjectUUID(ctx, projectUUID)
	require.NoError(t, err)
	require.Len(t, finalMembers, 2)
}
