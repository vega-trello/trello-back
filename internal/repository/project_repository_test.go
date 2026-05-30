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
	dto "github.com/vega-trello/trello-back/internal/dto/project"
)

func setupProjectRepo(t *testing.T) (*ProjectRepository, *pgxpool.Pool, uuid.UUID) {
	t.Helper()
	pool := setupTestPool(t)
	repo := NewProjectRepository(pool)

	creatorUUID := uuid.New()

	_, err := pool.Exec(context.Background(), `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, 'test_creator', 'manual', NOW(), NOW())
		ON CONFLICT (uuid) DO NOTHING
	`, creatorUUID)
	require.NoError(t, err)

	return repo, pool, creatorUUID
}

func strPtr(s string) *string {
	return &s
}

func TestProjectRepository_Create_Success(t *testing.T) {
	repo, pool, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	req := dto.CreateProjectRequest{
		Title:       "Test Project",
		Description: strPtr("Project description"),
	}

	project, err := repo.Create(ctx, creatorUUID, req)
	require.NoError(t, err)
	require.NotNil(t, project)

	assert.NotEmpty(t, project.UUID)
	assert.Equal(t, "Test Project", project.Title)
	assert.Equal(t, "Project description", *project.Description)
	assert.False(t, project.CreatedAt.IsZero())
	assert.False(t, project.UpdatedAt.IsZero())

	var roleID int
	err = pool.QueryRow(ctx, `
		SELECT role_id FROM project_member
		WHERE project_uuid = $1 AND user_uuid = $2
	`, project.UUID, creatorUUID).Scan(&roleID)
	require.NoError(t, err)
	assert.Equal(t, 1, roleID)
}

func TestProjectRepository_Create_WithoutDescription(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	req := dto.CreateProjectRequest{
		Title:       "No Desc Project",
		Description: nil,
	}

	project, err := repo.Create(ctx, creatorUUID, req)
	require.NoError(t, err)

	assert.Equal(t, "No Desc Project", project.Title)
	assert.Nil(t, project.Description)
}

func TestProjectRepository_FindByID_Success(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{
		Title:       "Find Me",
		Description: strPtr("Found"),
	})

	project, err := repo.FindByID(ctx, created.UUID)
	require.NoError(t, err)
	require.NotNil(t, project)

	assert.Equal(t, created.UUID, project.UUID)
	assert.Equal(t, "Find Me", project.Title)
	assert.Equal(t, "Found", *project.Description)
}

func TestProjectRepository_FindByID_NotFound(t *testing.T) {
	repo, _, _ := setupProjectRepo(t)
	ctx := context.Background()

	randomUUID := uuid.New()
	project, err := repo.FindByID(ctx, randomUUID)
	assert.ErrorIs(t, err, ErrProjectNotFound)
	assert.Nil(t, project)
}

func TestProjectRepository_FindByUser_Success(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	_, _ = repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{Title: "Project 1"})
	_, _ = repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{Title: "Project 2"})

	projects, err := repo.FindByUser(ctx, creatorUUID)
	require.NoError(t, err)
	require.Len(t, projects, 2)

	titles := []string{projects[0].Title, projects[1].Title}
	assert.Contains(t, titles, "Project 1")
	assert.Contains(t, titles, "Project 2")
}

func TestProjectRepository_Update_TitleOnly(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{
		Title:       "Old Title",
		Description: strPtr("Old description"),
	})

	updated, err := repo.Update(ctx, created.UUID, creatorUUID, strPtr("New Title"), nil)
	require.NoError(t, err)

	assert.Equal(t, "New Title", updated.Title)
	assert.Equal(t, "Old description", *updated.Description) // осталось без изменений
}

func TestProjectRepository_Update_DescriptionOnly(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{
		Title:       "Keep Title",
		Description: strPtr("Old desc"),
	})

	updated, err := repo.Update(ctx, created.UUID, creatorUUID, nil, strPtr("New desc"))
	require.NoError(t, err)

	assert.Equal(t, "Keep Title", updated.Title)      // осталось
	assert.Equal(t, "New desc", *updated.Description) // обновилось
}

func TestProjectRepository_Update_ClearDescription(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{
		Title:       "Test",
		Description: strPtr("To be cleared"),
	})

	empty := ""
	updated, err := repo.Update(ctx, created.UUID, creatorUUID, nil, &empty)
	require.NoError(t, err)

	assert.Equal(t, "Test", updated.Title)
	assert.Equal(t, "", *updated.Description) // теперь пустая строка, не nil
}

func TestProjectRepository_Update_BothFields(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{
		Title:       "Old",
		Description: strPtr("Old"),
	})

	updated, err := repo.Update(ctx, created.UUID, creatorUUID, strPtr("New Title"), strPtr("New Desc"))
	require.NoError(t, err)

	assert.Equal(t, "New Title", updated.Title)
	assert.Equal(t, "New Desc", *updated.Description)
}

func TestProjectRepository_Update_AccessDenied(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{Title: "Private"})

	otherUser := uuid.New()
	_, err := repo.Update(ctx, created.UUID, otherUser, strPtr("New"), nil)
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestProjectRepository_Update_NotFound(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	randomUUID := uuid.New()
	_, err := repo.Update(ctx, randomUUID, creatorUUID, strPtr("New"), nil)

	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestProjectRepository_Delete_Success(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{Title: "ToDelete"})

	err := repo.Delete(ctx, created.UUID)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, created.UUID)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestProjectRepository_Delete_NotFound(t *testing.T) {
	repo, _, _ := setupProjectRepo(t)
	ctx := context.Background()

	randomUUID := uuid.New()
	err := repo.Delete(ctx, randomUUID)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestProjectRepository_IsMember_Success(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	project, _ := repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{Title: "MemberTest"})

	isMember, err := repo.IsMember(ctx, project.UUID, creatorUUID)
	require.NoError(t, err)
	assert.True(t, isMember)

	otherUser := uuid.New()
	isMember, err = repo.IsMember(ctx, project.UUID, otherUser)
	require.NoError(t, err)
	assert.False(t, isMember)
}

func TestProjectRepository_IsOwner_Success(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	project, _ := repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{Title: "OwnerTest"})

	isOwner, err := repo.IsOwner(ctx, project.UUID, creatorUUID)
	require.NoError(t, err)
	assert.True(t, isOwner)

	otherUser := uuid.New()
	isOwner, err = repo.IsOwner(ctx, project.UUID, otherUser)
	require.NoError(t, err)
	assert.False(t, isOwner)
}

func TestProjectRepository_RemoveMember_Success(t *testing.T) {
	repo, pool, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	project, _ := repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{Title: "RemoveTest"})

	secondUser := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, 'second_user', 'manual', NOW(), NOW())
		ON CONFLICT (uuid) DO NOTHING
	`, secondUser)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO project_member (project_uuid, user_uuid, role_id, joined_at)
		VALUES ($1, $2, 3, NOW())
	`, project.UUID, secondUser)
	require.NoError(t, err)

	isMember, err := repo.IsMember(ctx, project.UUID, secondUser)
	require.NoError(t, err)
	assert.True(t, isMember)

	err = repo.RemoveMember(ctx, project.UUID, secondUser)
	assert.NoError(t, err)

	isMember, err = repo.IsMember(ctx, project.UUID, secondUser)
	require.NoError(t, err)
	assert.False(t, isMember)

	_, err = repo.FindByID(ctx, project.UUID)
	assert.NoError(t, err)
}

func TestProjectRepository_RemoveMember_Owner(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	project, _ := repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{Title: "RemoveOwnerTest"})

	isOwner, err := repo.IsOwner(ctx, project.UUID, creatorUUID)
	require.NoError(t, err)
	assert.True(t, isOwner)

	err = repo.RemoveMember(ctx, project.UUID, creatorUUID)
	assert.NoError(t, err)

	isMember, err := repo.IsMember(ctx, project.UUID, creatorUUID)
	require.NoError(t, err)
	assert.False(t, isMember)

	isOwner, err = repo.IsOwner(ctx, project.UUID, creatorUUID)
	require.NoError(t, err)
	assert.False(t, isOwner)

	_, err = repo.FindByID(ctx, project.UUID)
	assert.NoError(t, err)
}

func TestProjectRepository_RemoveMember_NotMember(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	project, _ := repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{Title: "NotMemberTest"})

	otherUser := uuid.New()
	err := repo.RemoveMember(ctx, project.UUID, otherUser)

	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestProjectRepository_RemoveMember_ProjectNotFound(t *testing.T) {
	repo, _, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	randomUUID := uuid.New()
	err := repo.RemoveMember(ctx, randomUUID, creatorUUID)

	assert.Error(t, err)
}

func TestProjectRepository_DeleteProject_FullWorkflow(t *testing.T) {
	repo, pool, creatorUUID := setupProjectRepo(t)
	ctx := context.Background()

	project, _ := repo.Create(ctx, creatorUUID, dto.CreateProjectRequest{Title: "WorkflowTest"})

	secondUser := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, 'second', 'manual', NOW(), NOW())
		ON CONFLICT (uuid) DO NOTHING
	`, secondUser)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO project_member (project_uuid, user_uuid, role_id, joined_at)
		VALUES ($1, $2, 3, NOW())
	`, project.UUID, secondUser)
	require.NoError(t, err)

	err = repo.Delete(ctx, project.UUID)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, project.UUID)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}
