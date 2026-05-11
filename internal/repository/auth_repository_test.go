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
	"golang.org/x/crypto/bcrypt"
)

func setupAuthRepo(t *testing.T) (*AuthRepository, *pgxpool.Pool) {
	t.Helper()
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	return repo, pool
}

func TestAuthRepository_RegisterPasswordUser_Success(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	user, err := repo.RegisterPasswordUser(ctx, "testuser", hashPassword(t, "pass123"))
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.NotEmpty(t, user.UUID)
	assert.Equal(t, "testuser", user.Username)
}

func TestAuthRepository_RegisterPasswordUser_DuplicateUsername(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	_, err := repo.RegisterPasswordUser(ctx, "duplicate", hashPassword(t, "pass1"))
	require.NoError(t, err)

	_, err = repo.RegisterPasswordUser(ctx, "duplicate", hashPassword(t, "pass2"))
	assert.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestAuthRepository_FindUserByUsername_Success(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	password := "secure_pass"
	_, err := repo.RegisterPasswordUser(ctx, "findme", hashPassword(t, password))
	require.NoError(t, err)

	user, hash, err := repo.FindUserByUsername(ctx, "findme")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, hash)

	assert.Equal(t, "findme", user.Username)
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	assert.NoError(t, err)
}

func TestAuthRepository_FindUserByUsername_InvalidCredentials(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	user, hash, err := repo.FindUserByUsername(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Nil(t, user)
	assert.Nil(t, hash)
}

func TestAuthRepository_FindOrCreateUserBySSO_NewUser(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	user, err := repo.FindOrCreateUserBySSO(ctx, "google", "ext_123", "sso_user")
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Equal(t, "sso_user", user.Username)
}

func TestAuthRepository_FindOrCreateUserBySSO_ExistingUser(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	_, err := repo.FindOrCreateUserBySSO(ctx, "github", "ext_456", "gh_user")
	require.NoError(t, err)

	user, err := repo.FindOrCreateUserBySSO(ctx, "github", "ext_456", "gh_user")
	require.NoError(t, err)
	assert.Equal(t, "gh_user", user.Username)
}

func TestAuthRepository_FindUserByUUID_Success(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	registered, _ := repo.RegisterPasswordUser(ctx, "byuuid", hashPassword(t, "pass"))

	user, err := repo.FindUserByUUID(ctx, registered.UUID)
	require.NoError(t, err)
	assert.Equal(t, registered.UUID, user.UUID)
	assert.Equal(t, "byuuid", user.Username)
}

func TestAuthRepository_FindUserByUUID_NotFound(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	randomUUID := uuid.New()
	user, err := repo.FindUserByUUID(ctx, randomUUID)
	assert.ErrorIs(t, err, ErrUserNotFound)
	assert.Nil(t, user)
}

func TestAuthRepository_UpdateUser_Username_Success(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	user, _ := repo.RegisterPasswordUser(ctx, "oldname", hashPassword(t, "pass"))

	newUsername := "newname"
	updated, err := repo.UpdateUser(ctx, user.UUID, &newUsername, nil)
	require.NoError(t, err)

	assert.Equal(t, "newname", updated.Username)
	assert.Equal(t, user.UUID, updated.UUID)
}

func TestAuthRepository_UpdateUser_Password_Success(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	user, _ := repo.RegisterPasswordUser(ctx, "passuser", hashPassword(t, "oldpass"))

	newHash := hashPassword(t, "newpass")
	updated, err := repo.UpdateUser(ctx, user.UUID, nil, newHash)
	require.NoError(t, err)
	require.NotNil(t, updated)

	_, hash, err := repo.FindUserByUsername(ctx, "passuser")
	require.NoError(t, err)
	err = bcrypt.CompareHashAndPassword(hash, []byte("newpass"))
	assert.NoError(t, err)
}

func TestAuthRepository_UpdateUser_SSO_CannotChangePassword(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	user, _ := repo.FindOrCreateUserBySSO(ctx, "oauth", "ext_789", "sso_no_pass")

	newHash := hashPassword(t, "attempt")
	_, err := repo.UpdateUser(ctx, user.UUID, nil, newHash)
	assert.ErrorIs(t, err, ErrSSOUserPasswordChange)
}

func TestAuthRepository_Logout_Success(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	user, _ := repo.RegisterPasswordUser(ctx, "logout_user", hashPassword(t, "pass"))

	err := repo.Logout(ctx, user.UUID)
	assert.NoError(t, err)
}
