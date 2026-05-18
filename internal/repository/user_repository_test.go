//go:build integration
// +build integration

package repository

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vega-trello/trello-back/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func truncateUsers(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE sso_user, manual_user, base_user RESTART IDENTITY CASCADE
	`)
	require.NoError(t, err, "failed to truncate user tables")
}

func TestUserRepository_RegisterPasswordUser_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	username := "testuser"
	passwordHash := hashPassword(t, "pass123")

	user, err := repo.RegisterPasswordUser(ctx, username, passwordHash)

	require.NoError(t, err)
	assert.NotEmpty(t, user.UUID)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, "manual", user.UserType)
}

func TestUserRepository_RegisterPasswordUser_DuplicateUsername(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	username := "duplicate"
	passwordHash := hashPassword(t, "pass1")

	_, err := repo.RegisterPasswordUser(ctx, username, passwordHash)
	require.NoError(t, err)

	_, err = repo.RegisterPasswordUser(ctx, username, hashPassword(t, "pass2"))
	assert.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestUserRepository_FindUserByUsername_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	username := "findme"
	password := "secure_pass"
	passwordHash := hashPassword(t, password)

	_, err := repo.RegisterPasswordUser(ctx, username, passwordHash)
	require.NoError(t, err)

	foundUser, foundHash, err := repo.FindUserByUsername(ctx, username)

	require.NoError(t, err)
	assert.Equal(t, username, foundUser.Username)
	assert.Equal(t, "manual", foundUser.UserType)
	assert.NotNil(t, foundHash)

	err = bcrypt.CompareHashAndPassword(foundHash, []byte(password))
	assert.NoError(t, err)
}

func TestUserRepository_FindUserByUsername_InvalidCredentials(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	user, hash, err := repo.FindUserByUsername(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Nil(t, user)
	assert.Nil(t, hash)
}

func TestUserRepository_FindUserByUsername_SSOUserIgnored(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	_, err := repo.FindOrCreateUserBySSO(ctx, "vega", "123", "ssouser", json.RawMessage(`{}`))
	require.NoError(t, err)

	_, _, err = repo.FindUserByUsername(ctx, "ssouser")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestUserRepository_FindOrCreateUserBySSO_NewUser(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	provider := "vega"
	externalID := "ext_123"
	username := "sso_user"
	metadata := json.RawMessage(`{"fir":"Иван","grn":"КМБО-02-22"}`)

	user, err := repo.FindOrCreateUserBySSO(ctx, provider, externalID, username, metadata)

	require.NoError(t, err)
	assert.NotEmpty(t, user.UUID)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, "sso", user.UserType)

	var sso model.SsoUser
	err = pool.QueryRow(ctx, `
		SELECT user_uuid, provider, external_id, metadata 
		FROM sso_user WHERE external_id = $1
	`, externalID).Scan(&sso.UserUUID, &sso.Provider, &sso.ExternalID, &sso.Metadata)
	require.NoError(t, err)

	assert.Equal(t, provider, sso.Provider)
	assert.Equal(t, externalID, sso.ExternalID)
	assert.JSONEq(t, string(metadata), string(sso.Metadata))
}

func TestUserRepository_FindOrCreateUserBySSO_ExistingUser(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	provider := "github"
	externalID := "ext_456"
	username := "gh_user"

	_, err := repo.FindOrCreateUserBySSO(ctx, provider, externalID, username, json.RawMessage(`{}`))
	require.NoError(t, err)

	foundUser, err := repo.FindOrCreateUserBySSO(ctx, provider, externalID, "other_username", nil)
	require.NoError(t, err)

	assert.Equal(t, username, foundUser.Username)
	assert.Equal(t, "sso", foundUser.UserType)
}

func TestUserRepository_FindOrCreateUserBySSO_EmptyMetadata(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	_, err := repo.FindOrCreateUserBySSO(ctx, "vega", "111", "user111", nil)
	require.NoError(t, err)

	var meta json.RawMessage
	err = pool.QueryRow(ctx, `
		SELECT metadata FROM sso_user WHERE external_id = '111'
	`).Scan(&meta)
	require.NoError(t, err)

	assert.JSONEq(t, `{}`, string(meta))
}

func TestUserRepository_FindOrCreateUserBySSO_UsernameConflict(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	_, err := repo.RegisterPasswordUser(ctx, "conflict", hashPassword(t, "hash"))
	require.NoError(t, err)

	_, err = repo.FindOrCreateUserBySSO(ctx, "vega", "999", "conflict", json.RawMessage(`{}`))

	assert.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestUserRepository_FindUserByUUID_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	registered, _ := repo.RegisterPasswordUser(ctx, "byuuid", hashPassword(t, "pass"))

	user, err := repo.FindUserByUUID(ctx, registered.UUID)
	require.NoError(t, err)
	assert.Equal(t, registered.UUID, user.UUID)
	assert.Equal(t, "byuuid", user.Username)
	assert.Equal(t, "manual", user.UserType)
}

func TestUserRepository_FindUserByUUID_NotFound(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	randomUUID := uuid.New()
	user, err := repo.FindUserByUUID(ctx, randomUUID)
	assert.ErrorIs(t, err, ErrUserNotFound)
	assert.Nil(t, user)
}

func TestUserRepository_GetSelfUser_Success_Manual(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	registered, _ := repo.RegisterPasswordUser(ctx, "selfmanual", hashPassword(t, "pass"))

	selfUser, err := repo.GetSelfUser(ctx, registered.UUID)
	require.NoError(t, err)
	require.NotNil(t, selfUser)

	assert.Equal(t, registered.UUID, selfUser.UUID)
	assert.Equal(t, "selfmanual", selfUser.Username)
	assert.Equal(t, "manual", selfUser.UserType)
	assert.False(t, selfUser.CreatedAt.IsZero())
	assert.False(t, selfUser.UpdatedAt.IsZero())
}

func TestUserRepository_GetSelfUser_Success_SSO(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	created, _ := repo.FindOrCreateUserBySSO(ctx, "vega", "777", "selfsso", json.RawMessage(`{}`))

	selfUser, err := repo.GetSelfUser(ctx, created.UUID)
	require.NoError(t, err)
	require.NotNil(t, selfUser)

	assert.Equal(t, "selfsso", selfUser.Username)
	assert.Equal(t, "sso", selfUser.UserType)
}

func TestUserRepository_GetSelfUser_NotFound(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	randomUUID := uuid.New()
	selfUser, err := repo.GetSelfUser(ctx, randomUUID)
	assert.ErrorIs(t, err, ErrUserNotFound)
	assert.Nil(t, selfUser)
}

func TestUserRepository_UpdateSelfUser_Manual_ChangeUsername(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	password := "oldpass123"
	user, _ := repo.RegisterPasswordUser(ctx, "oldname", hashPassword(t, password))

	updated, err := repo.UpdateSelfUser(ctx, user.UUID, password, "newname", "")
	require.NoError(t, err)
	require.NotNil(t, updated)

	assert.Equal(t, "newname", updated.Username)
	assert.Equal(t, "manual", updated.UserType)
}

func TestUserRepository_UpdateSelfUser_Manual_ChangePassword(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	oldPass := "oldpass"
	newPass := "newpass123"
	user, _ := repo.RegisterPasswordUser(ctx, "changepass", hashPassword(t, oldPass))

	_, err := repo.UpdateSelfUser(ctx, user.UUID, oldPass, "", newPass)
	require.NoError(t, err)

	err = repo.VerifyPassword(ctx, user.UUID, newPass)
	assert.NoError(t, err)

	_, hash, err := repo.FindUserByUsername(ctx, "changepass")
	require.NoError(t, err)
	err = bcrypt.CompareHashAndPassword(hash, []byte(oldPass))
	assert.Error(t, err)
}

func TestUserRepository_UpdateSelfUser_Manual_WrongOldPassword(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	user, _ := repo.RegisterPasswordUser(ctx, "test", hashPassword(t, "realpass"))

	_, err := repo.UpdateSelfUser(ctx, user.UUID, "wrongpass", "newname", "")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestUserRepository_UpdateSelfUser_SSO_ChangeUsername(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	user, _ := repo.FindOrCreateUserBySSO(ctx, "vega", "888", "ssoold", json.RawMessage(`{}`))

	updated, err := repo.UpdateSelfUser(ctx, user.UUID, "", "ssonew", "")
	require.NoError(t, err)
	require.NotNil(t, updated)

	assert.Equal(t, "ssonew", updated.Username)
	assert.Equal(t, "sso", updated.UserType)
}

func TestUserRepository_UpdateSelfUser_SSO_ChangePassword_Forbidden(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	user, _ := repo.FindOrCreateUserBySSO(ctx, "vega", "999", "ssonopass", json.RawMessage(`{}`))

	_, err := repo.UpdateSelfUser(ctx, user.UUID, "", "", "newpass")
	assert.ErrorIs(t, err, ErrSSOUserPasswordChange)
}

func TestUserRepository_UpdateSelfUser_Both_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	oldPass := "oldpass"
	newPass := "newpass"
	user, _ := repo.RegisterPasswordUser(ctx, "oldname", hashPassword(t, oldPass))

	updated, err := repo.UpdateSelfUser(ctx, user.UUID, oldPass, "newname", newPass)
	require.NoError(t, err)

	assert.Equal(t, "newname", updated.Username)

	_, hash, err := repo.FindUserByUsername(ctx, "newname")
	require.NoError(t, err)
	err = bcrypt.CompareHashAndPassword(hash, []byte(newPass))
	assert.NoError(t, err)
}

func TestUserRepository_UpdateSelfUser_DuplicateUsername(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	_, _ = repo.RegisterPasswordUser(ctx, "user1", hashPassword(t, "pass1"))
	user2, _ := repo.RegisterPasswordUser(ctx, "user2", hashPassword(t, "pass2"))

	_, err := repo.UpdateSelfUser(ctx, user2.UUID, "pass2", "user1", "")
	assert.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestUserRepository_VerifyPassword_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	password := "verify_pass"
	user, _ := repo.RegisterPasswordUser(ctx, "verify_user", hashPassword(t, password))

	err := repo.VerifyPassword(ctx, user.UUID, password)
	assert.NoError(t, err)
}

func TestUserRepository_VerifyPassword_Wrong(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	user, _ := repo.RegisterPasswordUser(ctx, "verify_user", hashPassword(t, "correct"))

	err := repo.VerifyPassword(ctx, user.UUID, "wrong")
	assert.Error(t, err)
}

func TestUserRepository_VerifyPassword_SSO_NotSupported(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	user, _ := repo.FindOrCreateUserBySSO(ctx, "oauth", "ext_999", "sso_verify", json.RawMessage(`{}`))

	err := repo.VerifyPassword(ctx, user.UUID, "any")
	assert.ErrorIs(t, err, ErrSSOUserPasswordChange)
}

func TestUserRepository_Logout_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	truncateUsers(t, pool)

	user, _ := repo.RegisterPasswordUser(ctx, "logout_user", hashPassword(t, "pass"))

	err := repo.Logout(ctx, user.UUID)
	assert.NoError(t, err)
}
