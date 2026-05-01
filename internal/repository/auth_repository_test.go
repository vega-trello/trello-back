//go:build integration
// +build integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthRepository_RegisterPasswordUser_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	ctx := context.Background()

	username := "testuser"
	hash := hashPassword(t, "securepassword")

	user, err := repo.RegisterPasswordUser(ctx, username, hash)

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, "manual", user.UserType)
	assert.NotEmpty(t, user.UUID)
}

func TestAuthRepository_RegisterPasswordUser_Duplicate(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	ctx := context.Background()

	username := "duplicate_user"
	hash := hashPassword(t, "password")

	_, err := repo.RegisterPasswordUser(ctx, username, hash)
	require.NoError(t, err)

	_, err = repo.RegisterPasswordUser(ctx, username, hash)
	require.Error(t, err)
	assert.Equal(t, ErrUserAlreadyExists, err)
}

func TestAuthRepository_FindUserByUsername_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	ctx := context.Background()

	username := "login_user"
	password := "secret"
	hash := hashPassword(t, password)

	_, err := repo.RegisterPasswordUser(ctx, username, hash)
	require.NoError(t, err)

	foundUser, foundHash, err := repo.FindUserByUsername(ctx, username)

	require.NoError(t, err)
	require.NotNil(t, foundUser)
	require.NotNil(t, foundHash)
	assert.Equal(t, username, foundUser.Username)
	assert.Equal(t, "manual", foundUser.UserType)
	assert.NotEmpty(t, foundHash)
}

func TestAuthRepository_FindUserByUsername_NotFound(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	ctx := context.Background()

	_, _, err := repo.FindUserByUsername(ctx, "non_existent_user")
	require.Error(t, err)
	assert.Equal(t, ErrInvalidCredentials, err)
}

func TestAuthRepository_FindUserByUsername_SSOUser(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	ctx := context.Background()

	// Создаём SSO-пользователя напрямую в БД
	userUUID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, 'sso', NOW(), NOW())
	`, userUUID, "sso_user")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO sso_user (user_uuid, provider, external_id, metadata)
		VALUES ($1, $2, $3, '{}')
	`, userUUID, "google", "12345")
	require.NoError(t, err)

	// Пытаемся найти через метод для manual-пользователей
	_, _, err = repo.FindUserByUsername(ctx, "sso_user")
	require.Error(t, err)
	assert.Equal(t, ErrInvalidCredentials, err)
}

func TestAuthRepository_FindUserByUUID_Success(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	ctx := context.Background()

	userUUID := createTestUser(t, pool, "uuid_user", "pass123")

	foundUser, err := repo.FindUserByUUID(ctx, userUUID)

	require.NoError(t, err)
	require.NotNil(t, foundUser)
	assert.Equal(t, "uuid_user", foundUser.Username)
	assert.Equal(t, userUUID, foundUser.UUID)
}

func TestAuthRepository_FindUserByUUID_NotFound(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	ctx := context.Background()

	fakeUUID := uuid.New()
	_, err := repo.FindUserByUUID(ctx, fakeUUID)

	require.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestAuthRepository_UpdateUser_Username(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	ctx := context.Background()

	userUUID := createTestUser(t, pool, "old_name", "pass123")
	newUsername := "new_name"

	updated, err := repo.UpdateUser(ctx, userUUID, &newUsername, nil)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "new_name", updated.Username)
}

func TestAuthRepository_UpdateUser_Password(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	ctx := context.Background()

	userUUID := createTestUser(t, pool, "pass_change", "old_pass")
	newHash := hashPassword(t, "new_pass")

	updated, err := repo.UpdateUser(ctx, userUUID, nil, newHash)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "pass_change", updated.Username)

	// Проверяем, что новый пароль работает
	_, foundHash, err := repo.FindUserByUsername(ctx, "pass_change")
	require.NoError(t, err)
	assert.Equal(t, newHash, foundHash)
}

func TestAuthRepository_UpdateUser_SSO_PasswordDenied(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	ctx := context.Background()

	// Создаём SSO-пользователя
	userUUID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, 'sso', NOW(), NOW())
	`, userUUID, "sso_no_pass")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO sso_user (user_uuid, provider, external_id, metadata)
		VALUES ($1, $2, $3, '{}')
	`, userUUID, "google", "67890")
	require.NoError(t, err)

	newHash := hashPassword(t, "fake_pass")
	_, err = repo.UpdateUser(ctx, userUUID, nil, newHash)

	require.Error(t, err)
	assert.Equal(t, ErrSSOUserPasswordChange, err)
}

func TestAuthRepository_FindOrCreateUserBySSO_NewUser(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	ctx := context.Background()

	provider := "google"
	externalID := "sso_12345"
	username := "sso_newbie"

	user, err := repo.FindOrCreateUserBySSO(ctx, provider, externalID, username)

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, "sso", user.UserType)
	assert.NotEmpty(t, user.UUID)

	// Проверяем, что запись создана в sso_user
	var ssoProvider, ssoExternalID string
	err = pool.QueryRow(ctx, `
		SELECT provider, external_id FROM sso_user WHERE user_uuid = $1
	`, user.UUID).Scan(&ssoProvider, &ssoExternalID)
	require.NoError(t, err)
	assert.Equal(t, provider, ssoProvider)
	assert.Equal(t, externalID, ssoExternalID)
}

func TestAuthRepository_FindOrCreateUserBySSO_ExistingUser(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	ctx := context.Background()

	provider := "github"
	externalID := "gh_999"
	username := "github_user"

	// Первый вызов - создаёт
	user1, err := repo.FindOrCreateUserBySSO(ctx, provider, externalID, username)
	require.NoError(t, err)

	// Второй вызов - находит существующего
	user2, err := repo.FindOrCreateUserBySSO(ctx, provider, externalID, username)
	require.NoError(t, err)

	// Должен вернуть того же пользователя
	assert.Equal(t, user1.UUID, user2.UUID)
	assert.Equal(t, username, user2.Username)
}

func TestAuthRepository_Logout_Stateless(t *testing.T) {
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)
	ctx := context.Background()

	userUUID := createTestUser(t, pool, "logout_user", "pass123")

	// Для stateless JWT logout - это no-op
	err := repo.Logout(ctx, userUUID)
	assert.NoError(t, err)
}
