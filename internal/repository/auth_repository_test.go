//go:build integration
// +build integration

package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// setupAuthRepo инициализирует репозиторий аутентификации
func setupAuthRepo(t *testing.T) (*AuthRepository, *pgxpool.Pool) {
	t.Helper()
	pool := setupTestPool(t)
	repo := NewAuthRepository(pool)

	t.Cleanup(func() {
		cleanAllTables(t, pool)
	})

	return repo, pool
}

// hashPassword хеширует пароль для тестов (используем bcrypt)
func hashPassword(t *testing.T, password string) []byte {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err, "failed to hash password")
	return hash
}

func TestAuthRepository_Register_Success(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	username := "testuser_" + uuid.New().String()[:8]
	passwordHash := hashPassword(t, "secure_password_123")

	user, err := repo.Register(ctx, username, passwordHash)
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.NotZero(t, user.UUID)
	assert.Equal(t, username, user.Username)
	assert.Equal(t, "manual", user.UserType)
	assert.WithinDuration(t, time.Now(), user.CreatedAt, 2*time.Second)
	assert.WithinDuration(t, user.CreatedAt, user.UpdatedAt, 1*time.Second)
}

func TestAuthRepository_Register_DuplicateUsername(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	username := "duplicate_user_" + uuid.New().String()[:8]
	passwordHash := hashPassword(t, "password1")

	// Первая регистрация успешна
	_, err := repo.Register(ctx, username, passwordHash)
	require.NoError(t, err)

	// Вторая регистрация с тем же username ошибка
	_, err = repo.Register(ctx, username, hashPassword(t, "password2"))
	assert.ErrorIs(t, err, ErrUserAlreadyExists)
}

func TestAuthRepository_LoginByUsername_Success(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	username := "login_user_" + uuid.New().String()[:8]
	password := "correct_password"
	passwordHash := hashPassword(t, password)

	// Сначала регистрируем пользователя
	_, err := repo.Register(ctx, username, passwordHash)
	require.NoError(t, err)

	// Затем логинимся
	user, returnedHash, err := repo.LoginByUsername(ctx, username)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotEmpty(t, returnedHash)

	assert.Equal(t, username, user.Username)
	assert.Equal(t, "manual", user.UserType)

	// Проверяем хеш через bcrypt, а не прямым сравнением
	err = bcrypt.CompareHashAndPassword(returnedHash, []byte(password))
	assert.NoError(t, err)
}

func TestAuthRepository_LoginByUsername_NotFound(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	// Пользователь не существует
	_, _, err := repo.LoginByUsername(ctx, "nonexistent_user")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthRepository_LoginByUsername_WrongType(t *testing.T) {
	repo, pool := setupAuthRepo(t)
	ctx := context.Background()

	// Создаём SSO пользователя напрямую (не через Register)
	userUUID := createTestBaseUser(t, pool)
	_, err := pool.Exec(ctx, `
		INSERT INTO sso_user (user_uuid, provider, external_id)
		VALUES ($1, $2, $3)
	`, userUUID, "google", "external_123")
	require.NoError(t, err)

	// Попытка логина по username для manual_user не найдёт SSO пользователя
	_, _, err = repo.LoginByUsername(ctx, "testuser_"+userUUID.String()[:8])
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthRepository_UpdatePassword_Success(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	username := "update_user_" + uuid.New().String()[:8]
	oldPassword := "old_password"
	newPassword := "new_password"

	// Регистрируем пользователя
	user, err := repo.Register(ctx, username, hashPassword(t, oldPassword))
	require.NoError(t, err)

	// Обновляем пароль
	err = repo.UpdatePassword(ctx, user.UUID, hashPassword(t, newPassword))
	assert.NoError(t, err)

	// Проверяем, что новый пароль сохранился
	_, returnedHash, err := repo.LoginByUsername(ctx, username)
	require.NoError(t, err)
	err = bcrypt.CompareHashAndPassword(returnedHash, []byte(newPassword))
	assert.NoError(t, err)
}
func TestAuthRepository_UpdatePassword_UpdatesTimestamp(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	username := "timestamp_user_" + uuid.New().String()[:8]
	passwordHash := hashPassword(t, "password")

	user, err := repo.Register(ctx, username, passwordHash)
	require.NoError(t, err)
	createdAt := user.CreatedAt
	updatedAt := user.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	err = repo.UpdatePassword(ctx, user.UUID, hashPassword(t, "new_password"))
	require.NoError(t, err)

	updatedUser, err := repo.FindByUsername(ctx, username)
	require.NoError(t, err)

	assert.Equal(t, createdAt, updatedUser.CreatedAt)
	assert.True(t, updatedUser.UpdatedAt.After(updatedAt))
}

func TestAuthRepository_UpdatePassword_UserNotFound(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	nonExistentUUID := uuid.New()
	err := repo.UpdatePassword(ctx, nonExistentUUID, hashPassword(t, "new_password"))
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestAuthRepository_Logout_Stateless(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	// Stateless JWT: logout — пустая операция
	// Просто проверяем, что метод не паникует и возвращает nil
	userUUID := uuid.New()
	err := repo.Logout(ctx, userUUID)
	assert.NoError(t, err)
}

func TestAuthRepository_FindByUsername_Success(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	username := "find_user_" + uuid.New().String()[:8]
	passwordHash := hashPassword(t, "password")

	// Регистрируем пользователя
	original, err := repo.Register(ctx, username, passwordHash)
	require.NoError(t, err)

	// Находим по username
	found, err := repo.FindByUsername(ctx, username)
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, original.UUID, found.UUID)
	assert.Equal(t, original.Username, found.Username)
	assert.Equal(t, original.UserType, found.UserType)
	assert.Equal(t, original.CreatedAt, found.CreatedAt)
	assert.Equal(t, original.UpdatedAt, found.UpdatedAt)
}

func TestAuthRepository_FindByUsername_NotFound(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	_, err := repo.FindByUsername(ctx, "nonexistent_user")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestAuthRepository_FindByUsername_CaseSensitive(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	username := "CaseSensitiveUser"
	passwordHash := hashPassword(t, "password")

	// Регистрируем с определённым регистром
	_, err := repo.Register(ctx, username, passwordHash)
	require.NoError(t, err)

	// Поиск с другим регистром не должен найти (если в БД нет COLLATE case-insensitive)
	_, err = repo.FindByUsername(ctx, "casesensitiveuser")
	// Поведение зависит от настроек БД, но обычно это ошибка
	assert.Error(t, err)
}

func TestAuthRepository_FullWorkflow(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	username := "workflow_user_" + uuid.New().String()[:8]
	originalPassword := "original_pass"
	newPassword := "new_pass"

	// Регистрация
	createdUser, err := repo.Register(ctx, username, hashPassword(t, originalPassword))
	require.NoError(t, err)
	assert.Equal(t, "manual", createdUser.UserType)

	// Логин с оригинальным паролем
	_, hash1, err := repo.LoginByUsername(ctx, username)
	require.NoError(t, err)
	// Проверяем хеш через bcrypt, а не прямым сравнением
	err = bcrypt.CompareHashAndPassword(hash1, []byte(originalPassword))
	assert.NoError(t, err)

	// Обновление пароля
	err = repo.UpdatePassword(ctx, createdUser.UUID, hashPassword(t, newPassword))
	require.NoError(t, err)

	// Логин с новым паролем
	_, hash2, err := repo.LoginByUsername(ctx, username)
	require.NoError(t, err)
	// Проверяем новый пароль через bcrypt
	err = bcrypt.CompareHashAndPassword(hash2, []byte(newPassword))
	assert.NoError(t, err)

	// Поиск пользователя
	foundUser, err := repo.FindByUsername(ctx, username)
	require.NoError(t, err)
	assert.Equal(t, createdUser.UUID, foundUser.UUID)

	// Logout (stateless)
	err = repo.Logout(ctx, createdUser.UUID)
	assert.NoError(t, err)
}

func TestAuthRepository_Register_LongUsername(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	// Username длиной 64 символа (максимум по схеме)
	username := strings.Repeat("a", 64)
	passwordHash := hashPassword(t, "password")

	user, err := repo.Register(ctx, username, passwordHash)
	require.NoError(t, err)
	assert.Equal(t, username, user.Username)
}

func TestAuthRepository_Register_UsernameTooLong(t *testing.T) {
	repo, _ := setupAuthRepo(t)
	ctx := context.Background()

	// Username длиной 65 символов (превышает лимит 64)
	username := strings.Repeat("a", 65)
	passwordHash := hashPassword(t, "password")

	_, err := repo.Register(ctx, username, passwordHash)
	// Ожидаем ошибку от БД: value too long for type character varying(64)
	assert.Error(t, err)
}
