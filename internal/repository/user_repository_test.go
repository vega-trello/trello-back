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

func setupUserRepo(t *testing.T) (*UserRepository, *pgxpool.Pool) {
	t.Helper()
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)
	return repo, pool
}

func TestUserRepository_RegisterPasswordUser_Success(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	user, err := repo.RegisterPasswordUser(ctx, "testuser", hashPassword(t, "pass123"))
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.NotEmpty(t, user.UUID)
	assert.Equal(t, "testuser", user.Username)
}

func TestUserRepository_RegisterPasswordUser_DuplicateUsername(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	_, err := repo.RegisterPasswordUser(ctx, "duplicate", hashPassword(t, "pass1"))
	require.NoError(t, err)

	_, err = repo.RegisterPasswordUser(ctx, "duplicate", hashPassword(t, "pass2"))
	assert.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestUserRepository_FindUserByUsername_Success(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	password := "secure_pass"
	_, err := repo.RegisterPasswordUser(ctx, "findme", hashPassword(t, password))
	require.NoError(t, err)

	user, hash, err := repo.FindUserByUsername(ctx, "findme")
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, hash)

	assert.Equal(t, "findme", user.Username)
	// Проверяем, что хеш соответствует паролю
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	assert.NoError(t, err)
}

func TestUserRepository_FindUserByUsername_InvalidCredentials(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	user, hash, err := repo.FindUserByUsername(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Nil(t, user)
	assert.Nil(t, hash)
}

func TestUserRepository_FindOrCreateUserBySSO_NewUser(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	user, err := repo.FindOrCreateUserBySSO(ctx, "google", "ext_123", "sso_user")
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Equal(t, "sso_user", user.Username)
}

func TestUserRepository_FindOrCreateUserBySSO_ExistingUser(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	// Создаём впервые
	_, err := repo.FindOrCreateUserBySSO(ctx, "github", "ext_456", "gh_user")
	require.NoError(t, err)

	// Находим существующего
	user, err := repo.FindOrCreateUserBySSO(ctx, "github", "ext_456", "gh_user")
	require.NoError(t, err)
	assert.Equal(t, "gh_user", user.Username)
}

func TestUserRepository_FindUserByUUID_Success(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	registered, _ := repo.RegisterPasswordUser(ctx, "byuuid", hashPassword(t, "pass"))

	user, err := repo.FindUserByUUID(ctx, registered.UUID)
	require.NoError(t, err)
	assert.Equal(t, registered.UUID, user.UUID)
	assert.Equal(t, "byuuid", user.Username)
}

func TestUserRepository_FindUserByUUID_NotFound(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	randomUUID := uuid.New()
	user, err := repo.FindUserByUUID(ctx, randomUUID)
	assert.ErrorIs(t, err, ErrUserNotFound)
	assert.Nil(t, user)
}

func TestUserRepository_GetSelfUser_Success(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	registered, _ := repo.RegisterPasswordUser(ctx, "selfuser", hashPassword(t, "pass"))

	selfUser, err := repo.GetSelfUser(ctx, registered.UUID)
	require.NoError(t, err)
	require.NotNil(t, selfUser)

	assert.Equal(t, registered.UUID, selfUser.UUID)
	assert.Equal(t, "selfuser", selfUser.Username)
	assert.Equal(t, "manual", selfUser.UserType)
	assert.False(t, selfUser.CreatedAt.IsZero())
	assert.False(t, selfUser.UpdatedAt.IsZero())
}

func TestUserRepository_GetSelfUser_NotFound(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	randomUUID := uuid.New()
	selfUser, err := repo.GetSelfUser(ctx, randomUUID)
	assert.ErrorIs(t, err, ErrUserNotFound)
	assert.Nil(t, selfUser)
}

func TestUserRepository_UpdateSelfUser_Username_Success(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	password := "oldpass"
	user, _ := repo.RegisterPasswordUser(ctx, "oldname", hashPassword(t, password))

	updated, err := repo.UpdateSelfUser(ctx, user.UUID, password, "newname", "")
	require.NoError(t, err)
	require.NotNil(t, updated)

	assert.Equal(t, "newname", updated.Username)
	assert.Equal(t, user.UUID, updated.UUID)
	assert.Equal(t, "manual", updated.UserType)
}

func TestUserRepository_UpdateSelfUser_Password_Success(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	oldPass := "oldpass"
	newPass := "newpass"
	user, _ := repo.RegisterPasswordUser(ctx, "passuser", hashPassword(t, oldPass))

	updated, err := repo.UpdateSelfUser(ctx, user.UUID, oldPass, "", newPass)
	require.NoError(t, err)
	require.NotNil(t, updated)

	// Проверяем, что новый пароль работает
	_, hash, err := repo.FindUserByUsername(ctx, "passuser")
	require.NoError(t, err)
	err = bcrypt.CompareHashAndPassword(hash, []byte(newPass))
	assert.NoError(t, err)

	// Старый пароль больше не работает
	_, hash, err = repo.FindUserByUsername(ctx, "passuser")
	require.NoError(t, err)
	err = bcrypt.CompareHashAndPassword(hash, []byte(oldPass))
	assert.Error(t, err)
}

func TestUserRepository_UpdateSelfUser_Both_Success(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	oldPass := "oldpass"
	newPass := "newpass"
	user, _ := repo.RegisterPasswordUser(ctx, "oldname", hashPassword(t, oldPass))

	updated, err := repo.UpdateSelfUser(ctx, user.UUID, oldPass, "newname", newPass)
	require.NoError(t, err)

	assert.Equal(t, "newname", updated.Username)

	// Проверяем новый пароль
	_, hash, err := repo.FindUserByUsername(ctx, "newname")
	require.NoError(t, err)
	err = bcrypt.CompareHashAndPassword(hash, []byte(newPass))
	assert.NoError(t, err)
}

func TestUserRepository_UpdateSelfUser_WrongOldPassword(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	user, _ := repo.RegisterPasswordUser(ctx, "testuser", hashPassword(t, "correctpass"))

	_, err := repo.UpdateSelfUser(ctx, user.UUID, "wrongpass", "newname", "newpass")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestUserRepository_UpdateSelfUser_SSO_CannotChangePassword(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	user, _ := repo.FindOrCreateUserBySSO(ctx, "oauth", "ext_789", "sso_no_pass")

	// SSO-пользователи не могут менять пароль через этот эндпоинт
	_, err := repo.UpdateSelfUser(ctx, user.UUID, "", "", "attempt")
	assert.ErrorIs(t, err, ErrSSOUserPasswordChange)
}

func TestUserRepository_UpdateSelfUser_SSO_CanChangeUsername(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	user, _ := repo.FindOrCreateUserBySSO(ctx, "oauth", "ext_789", "sso_old")

	// SSO-пользователи могут менять username (old_password не требуется для SSO)
	updated, err := repo.UpdateSelfUser(ctx, user.UUID, "", "sso_new", "")
	require.NoError(t, err)
	assert.Equal(t, "sso_new", updated.Username)
}

func TestUserRepository_UpdateSelfUser_DuplicateUsername(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	// Создаём двух пользователей
	_, _ = repo.RegisterPasswordUser(ctx, "user1", hashPassword(t, "pass1"))
	user2, _ := repo.RegisterPasswordUser(ctx, "user2", hashPassword(t, "pass2"))

	// Пытаемся переименовать user2 в user1
	_, err := repo.UpdateSelfUser(ctx, user2.UUID, "pass2", "user1", "")
	assert.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestUserRepository_VerifyPassword_Success(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	password := "verify_pass"
	user, _ := repo.RegisterPasswordUser(ctx, "verify_user", hashPassword(t, password))

	err := repo.VerifyPassword(ctx, user.UUID, password)
	assert.NoError(t, err)
}

func TestUserRepository_VerifyPassword_Wrong(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	user, _ := repo.RegisterPasswordUser(ctx, "verify_user", hashPassword(t, "correct"))

	err := repo.VerifyPassword(ctx, user.UUID, "wrong")
	assert.Error(t, err)
}

func TestUserRepository_VerifyPassword_SSO_NotSupported(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	user, _ := repo.FindOrCreateUserBySSO(ctx, "oauth", "ext_999", "sso_verify")

	err := repo.VerifyPassword(ctx, user.UUID, "any")
	assert.ErrorIs(t, err, ErrSSOUserPasswordChange)
}

func TestUserRepository_Logout_Success(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	user, _ := repo.RegisterPasswordUser(ctx, "logout_user", hashPassword(t, "pass"))

	err := repo.Logout(ctx, user.UUID)
	assert.NoError(t, err)
	// Stateless JWT: logout всегда успешен
}
