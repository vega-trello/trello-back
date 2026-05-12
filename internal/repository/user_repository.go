// internal/repository/user_repository.go
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vega-trello/trello-back/internal/model"
	"golang.org/x/crypto/bcrypt"
)

// UserRepositoryInterface определяет контракт для работы с пользователями и аутентификацией
type UserRepositoryInterface interface {
	// Auth endpoints
	RegisterPasswordUser(ctx context.Context, username string, passwordHash []byte) (*model.User, error)
	FindUserByUsername(ctx context.Context, username string) (*model.User, []byte, error)
	FindOrCreateUserBySSO(ctx context.Context, provider string, externalID string, username string) (*model.User, error)
	FindUserByUUID(ctx context.Context, userUUID uuid.UUID) (*model.User, error)

	// User endpoints (возвращают *model.SelfUser)
	GetSelfUser(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error)
	UpdateSelfUser(ctx context.Context, userUUID uuid.UUID, oldPassword string, newUsername string, newPassword string) (*model.SelfUser, error)

	// Вспомогательные
	VerifyPassword(ctx context.Context, userUUID uuid.UUID, password string) error
	Logout(ctx context.Context, userUUID uuid.UUID) error
}

// UserRepository реализует UserRepositoryInterface
type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// RegisterPasswordUser регистрирует нового manual-пользователя
func (r *UserRepository) RegisterPasswordUser(
	ctx context.Context,
	username string,
	passwordHash []byte,
) (*model.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	userUUID := uuid.New()

	// Создаём запись в base_user
	_, err = tx.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, 'manual', NOW(), NOW())
	`, userUUID, username)
	if err != nil {
		if isUniqueViolation(err, "base_user_username_key") {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("repository: create base_user: %w", err)
	}

	// Создаём запись в manual_user с хешем пароля
	_, err = tx.Exec(ctx, `
		INSERT INTO manual_user (user_uuid, password_hash)
		VALUES ($1, $2)
	`, userUUID, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("repository: create manual_user: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return &model.User{
		UUID:     userUUID,
		Username: username,
	}, nil
}

// FindUserByUsername находит manual-пользователя и возвращает хеш пароля для проверки
func (r *UserRepository) FindUserByUsername(
	ctx context.Context,
	username string,
) (*model.User, []byte, error) {
	var user model.User
	var passwordHash []byte

	err := r.db.QueryRow(ctx, `
		SELECT u.uuid, u.username, m.password_hash
		FROM base_user u
		JOIN manual_user m ON u.uuid = m.user_uuid
		WHERE u.username = $1 AND u.user_type = 'manual'
	`, username).Scan(&user.UUID, &user.Username, &passwordHash)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, nil, fmt.Errorf("repository: find user by username: %w", err)
	}

	return &user, passwordHash, nil
}

// FindOrCreateUserBySSO находит или создаёт SSO-пользователя (JIT Provisioning)
func (r *UserRepository) FindOrCreateUserBySSO(
	ctx context.Context,
	provider string,
	externalID string,
	username string,
) (*model.User, error) {
	var user model.User

	// Пытаемся найти существующего
	err := r.db.QueryRow(ctx, `
		SELECT u.uuid, u.username
		FROM base_user u
		JOIN sso_user s ON u.uuid = s.user_uuid
		WHERE s.provider = $1 AND s.external_id = $2
	`, provider, externalID).Scan(&user.UUID, &user.Username)

	if err == nil {
		return &user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("repository: check sso user: %w", err)
	}

	// Не найден - создаём нового
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	newUUID := uuid.New()

	_, err = tx.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, 'sso', NOW(), NOW())
	`, newUUID, username)
	if err != nil {
		if isUniqueViolation(err, "base_user_username_key") {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("repository: create sso base_user: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO sso_user (user_uuid, provider, external_id, metadata)
		VALUES ($1, $2, $3, '{}')
	`, newUUID, provider, externalID)
	if err != nil {
		return nil, fmt.Errorf("repository: create sso_user record: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return &model.User{
		UUID:     newUUID,
		Username: username,
	}, nil
}

// FindUserByUUID находит пользователя по UUID (для валидации JWT)
// Возвращает базовую модель *model.User
func (r *UserRepository) FindUserByUUID(
	ctx context.Context,
	userUUID uuid.UUID,
) (*model.User, error) {
	var user model.User

	err := r.db.QueryRow(ctx, `
		SELECT uuid, username
		FROM base_user WHERE uuid = $1
	`, userUUID).Scan(&user.UUID, &user.Username)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find user by uuid: %w", err)
	}

	return &user, nil
}

// GetSelfUser возвращает расширенную информацию о пользователе для эндпоинта GET /user
// Возвращает *model.SelfUser с полями: uuid, username, created_at, updated_at, user_type
func (r *UserRepository) GetSelfUser(
	ctx context.Context,
	userUUID uuid.UUID,
) (*model.SelfUser, error) {
	var user model.SelfUser

	err := r.db.QueryRow(ctx, `
		SELECT uuid, username, created_at, updated_at, user_type
		FROM base_user
		WHERE uuid = $1
	`, userUUID).Scan(
		&user.UUID, &user.Username, &user.CreatedAt, &user.UpdatedAt, &user.UserType,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: get self user: %w", err)
	}

	return &user, nil
}

// UpdateSelfUser обновляет профиль текущего пользователя для эндпоинта PATCH /user
// Требуется старый пароль, новое имя и/или новый пароль
// Возвращает обновлённый *model.SelfUser
func (r *UserRepository) UpdateSelfUser(
	ctx context.Context,
	userUUID uuid.UUID,
	oldPassword string,
	newUsername string,
	newPassword string,
) (*model.SelfUser, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	//  Получаем тип пользователя
	var userType, currentUsername string
	var currentHash []byte
	err = tx.QueryRow(ctx, `
		SELECT u.user_type, u.username, m.password_hash
		FROM base_user u
		LEFT JOIN manual_user m ON u.uuid = m.user_uuid
		WHERE u.uuid = $1
	`, userUUID).Scan(&userType, &currentUsername, &currentHash)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find user: %w", err)
	}

	// Проверяем старый пароль (только для manual-пользователей)
	if userType == "manual" {
		if err := bcrypt.CompareHashAndPassword(currentHash, []byte(oldPassword)); err != nil {
			return nil, ErrInvalidCredentials
		}
	} else {
		// SSO-пользователи не могут менять пароль через этот эндпоинт
		// Но могут менять username, если old_password совпадает (бизнес-правило: можно проверить через внешний провайдер)
		// Для простоты: запрещаем обновление пароля для SSO
		if newPassword != "" {
			return nil, ErrSSOUserPasswordChange
		}
	}

	// Обновляем username, если он изменился
	if newUsername != "" && newUsername != currentUsername {
		_, err = tx.Exec(ctx, `
			UPDATE base_user SET username = $1, updated_at = NOW()
			WHERE uuid = $2
		`, newUsername, userUUID)
		if err != nil {
			if isUniqueViolation(err, "base_user_username_key") {
				return nil, ErrUserAlreadyExists
			}
			return nil, fmt.Errorf("repository: update username: %w", err)
		}
	}

	// Обновляем пароль, если он передан (только для manual)
	if newPassword != "" && userType == "manual" {
		newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("repository: hash password: %w", err)
		}
		_, err = tx.Exec(ctx, `
			UPDATE manual_user SET password_hash = $1
			WHERE user_uuid = $2
		`, newHash, userUUID)
		if err != nil {
			return nil, fmt.Errorf("repository: update password: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	//  Возвращаем обновлённый профиль
	return r.GetSelfUser(ctx, userUUID)
}

// VerifyPassword проверяет пароль для manual-пользователя
// Используется внутри сервиса для дополнительной валидации
func (r *UserRepository) VerifyPassword(
	ctx context.Context,
	userUUID uuid.UUID,
	password string,
) error {
	var userType string
	var passwordHash []byte

	err := r.db.QueryRow(ctx, `
		SELECT u.user_type, m.password_hash
		FROM base_user u
		LEFT JOIN manual_user m ON u.uuid = m.user_uuid
		WHERE u.uuid = $1
	`, userUUID).Scan(&userType, &passwordHash)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("repository: fetch password: %w", err)
	}

	if userType != "manual" {
		return ErrSSOUserPasswordChange
	}

	return bcrypt.CompareHashAndPassword(passwordHash, []byte(password))
}

// Logout - заглушка для stateless JWT (инвалидация на клиенте)
func (r *UserRepository) Logout(ctx context.Context, userUUID uuid.UUID) error {
	// Stateless JWT: нет сессии на сервере для инвалидации
	// В будущем можно добавить blacklist токенов в Redis
	return nil
}

// isUniqueViolation проверяет, является ли ошибкой нарушение UNIQUE-ограничения
func isUniqueViolation(err error, constraintName string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" && pgErr.ConstraintName == constraintName
	}
	return false
}
