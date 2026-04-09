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
	"golang.org/x/crypto/bcrypt"
)

func setupUserRepo(t *testing.T) (*UserRepository, *pgxpool.Pool) {
	t.Helper()
	pool := setupTestPool(t)
	repo := NewUserRepository(pool)

	t.Cleanup(func() {
		cleanAllTables(t, pool)
	})

	return repo, pool
}

func TestUserRepository_FindByID_Success(t *testing.T) {
	repo, pool := setupUserRepo(t)
	ctx := context.Background()

	//Создаем пользователя напрямую (через SQL) для простоты
	userUUID := uuid.New()
	now := time.Now()
	_, err := pool.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
	`, userUUID, "findme", "manual", now)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, userUUID)
	require.NoError(t, err)

	assert.Equal(t, userUUID, found.UUID)
	assert.Equal(t, "findme", found.Username)
	assert.Equal(t, "manual", found.UserType)
	assert.WithinDuration(t, now, found.CreatedAt, 2*time.Second)
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepository_Update_UsernameSuccess(t *testing.T) {
	repo, pool := setupUserRepo(t)
	ctx := context.Background()

	userUUID := uuid.New()
	now := time.Now()
	_, err := pool.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
	`, userUUID, "oldname", "manual", now)
	require.NoError(t, err)

	// Обновляем имя
	newName := "newname"
	updated, err := repo.Update(ctx, userUUID, &newName, nil)
	require.NoError(t, err)

	assert.Equal(t, "newname", updated.Username)
	assert.True(t, updated.UpdatedAt.After(now), "updated_at должен измениться")
}

func TestUserRepository_Update_DuplicateUsername(t *testing.T) {
	repo, pool := setupUserRepo(t)
	ctx := context.Background()

	// Создаем первого пользователя с именем "taken"
	u1 := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, u1, "taken", "manual")
	require.NoError(t, err)

	// Создаем второго пользователя с именем "original"
	u2 := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, u2, "original", "manual")
	require.NoError(t, err)

	// Пытаемся изменить имя второго пользователя на "taken"
	duplicateName := "taken"
	_, err = repo.Update(ctx, u2, &duplicateName, nil)

	// Ожидаем ошибку уникальности
	assert.ErrorIs(t, err, ErrUsernameTaken)
}

func TestUserRepository_Update_PasswordSuccess_ManualUser(t *testing.T) {
	repo, pool := setupUserRepo(t)
	ctx := context.Background()

	// Создаем manual-пользователя через AuthRepository (так удобнее, создает 2 записи)
	authRepo := NewAuthRepository(pool)
	hash, _ := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.DefaultCost)
	createdUser, err := authRepo.Register(ctx, "manual_pass_user", hash)
	require.NoError(t, err)

	// Обновляем пароль
	newHash, _ := bcrypt.GenerateFromPassword([]byte("newpass"), bcrypt.DefaultCost)
	updated, err := repo.Update(ctx, createdUser.UUID, nil, newHash)
	require.NoError(t, err)

	// Проверяем, что returned_at обновился (или просто что нет ошибки)
	assert.Equal(t, createdUser.UUID, updated.UUID)
	assert.True(t, updated.UpdatedAt.After(createdUser.CreatedAt))
}

func TestUserRepository_Update_PasswordFail_SSOUser(t *testing.T) {
	repo, pool := setupUserRepo(t)
	ctx := context.Background()

	// Создаем SSO-пользователя (base_user + sso_user)
	uUID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, uUID, "sso_guy", "sso")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO sso_user (user_uuid, provider, external_id)
		VALUES ($1, $2, $3)
	`, uUID, "google", "external_123")
	require.NoError(t, err)

	newHash, _ := bcrypt.GenerateFromPassword([]byte("newpass"), bcrypt.DefaultCost)
	_, err = repo.Update(ctx, uUID, nil, newHash)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot update password")
}

func TestUserRepository_Update_UserNotFound(t *testing.T) {
	repo, _ := setupUserRepo(t)
	ctx := context.Background()

	newHash, _ := bcrypt.GenerateFromPassword([]byte("newpass"), bcrypt.DefaultCost)
	_, err := repo.Update(ctx, uuid.New(), nil, newHash)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserRepository_Update_NoChanges(t *testing.T) {
	repo, pool := setupUserRepo(t)
	ctx := context.Background()

	userUUID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, userUUID, "nochange", "manual")
	require.NoError(t, err)

	updated, err := repo.Update(ctx, userUUID, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "nochange", updated.Username)
}
