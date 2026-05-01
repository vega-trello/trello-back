package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vega-trello/trello-back/internal/model"
)

// AuthRepositoryInterface определяет контракт для работы с аутентификацией
type AuthRepositoryInterface interface {
	RegisterPasswordUser(ctx context.Context, username string, passwordHash []byte) (*model.User, error)
	FindUserByUsername(ctx context.Context, username string) (*model.User, []byte, error)
	FindOrCreateUserBySSO(ctx context.Context, provider string, externalID string, username string) (*model.User, error)
	FindUserByUUID(ctx context.Context, userUUID uuid.UUID) (*model.User, error)
	UpdateUser(ctx context.Context, userUUID uuid.UUID, username *string, newPasswordHash []byte) (*model.User, error)
	Logout(ctx context.Context, userUUID uuid.UUID) error
}

// AuthRepository реализует AuthRepositoryInterface с использованием pgxpool
type AuthRepository struct {
	db *pgxpool.Pool
}

// NewAuthRepository создает новый экземпляр репозитория
func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

// RegisterPasswordUser создает нового manual пользователя (base_user + manual_user)
func (r *AuthRepository) RegisterPasswordUser(
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
	now := time.Now()

	_, err = tx.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, 'manual', $3, $3)
	`, userUUID, username, now)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("repository: create base_user: %w", err)
	}

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
		UUID:      userUUID,
		Username:  username,
		UserType:  "manual",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// FindUserByUsername находит пользователя по имени и возвращает хеш пароля для проверки
// Работает только для manual пользователей (user_type = 'manual')
func (r *AuthRepository) FindUserByUsername(
	ctx context.Context,
	username string,
) (*model.User, []byte, error) {
	var user model.User
	var passwordHash []byte

	err := r.db.QueryRow(ctx, `
		SELECT u.uuid, u.username, u.user_type, u.created_at, u.updated_at, m.password_hash
		FROM base_user u
		JOIN manual_user m ON u.uuid = m.user_uuid
		WHERE u.username = $1 AND u.user_type = 'manual'
	`, username).Scan(
		&user.UUID,
		&user.Username,
		&user.UserType,
		&user.CreatedAt,
		&user.UpdatedAt,
		&passwordHash,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, nil, fmt.Errorf("repository: find user by username: %w", err)
	}

	return &user, passwordHash, nil
}

// FindOrCreateUserBySSO ищет пользователя по SSO provider+external_id.
// Если не находит - создает нового (JIT Provisioning).
func (r *AuthRepository) FindOrCreateUserBySSO(
	ctx context.Context,
	provider string,
	externalID string,
	username string,
) (*model.User, error) {

	var user model.User
	err := r.db.QueryRow(ctx, `
		SELECT u.uuid, u.username, u.user_type, u.created_at, u.updated_at
		FROM base_user u
		JOIN sso_user s ON u.uuid = s.user_uuid
		WHERE s.provider = $1 AND s.external_id = $2
	`, provider, externalID).Scan(
		&user.UUID, &user.Username, &user.UserType, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == nil {
		return &user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("repository: check sso user: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	newUUID := uuid.New()
	now := time.Now()

	_, err = tx.Exec(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, 'sso', $3, $3)
	`, newUUID, username, now)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("repository: create sso base_user: %w", err)
	}

	// Создаем запись в sso_user
	// metadata пока ставим пустым объектом '{}'
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
		UUID:      newUUID,
		Username:  username,
		UserType:  "sso",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// FindUserByUUID находит пользователя по UUID (используется для валидации JWT)
func (r *AuthRepository) FindUserByUUID(
	ctx context.Context,
	userUUID uuid.UUID,
) (*model.User, error) {
	var user model.User
	err := r.db.QueryRow(ctx, `
		SELECT uuid, username, user_type, created_at, updated_at
		FROM base_user WHERE uuid = $1
	`, userUUID).Scan(
		&user.UUID, &user.Username, &user.UserType, &user.CreatedAt, &user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find user by uuid: %w", err)
	}
	return &user, nil
}

// UpdateUser обновляет username и/или password_hash.
// newPasswordHash должен быть уже захеширован на уровне сервиса.
func (r *AuthRepository) UpdateUser(
	ctx context.Context,
	userUUID uuid.UUID,
	username *string,
	newPasswordHash []byte,
) (*model.User, error) {

	var currentUserType string
	err := r.db.QueryRow(ctx, `SELECT user_type FROM base_user WHERE uuid = $1`, userUUID).Scan(&currentUserType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if newPasswordHash != nil && currentUserType == "sso" {
		return nil, ErrSSOUserPasswordChange
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if username != nil {
		_, err = tx.Exec(ctx, `
			UPDATE base_user SET username = $1, updated_at = NOW()
			WHERE uuid = $2
		`, *username, userUUID)
		if err != nil {
			if IsUniqueViolation(err) {
				return nil, ErrUserAlreadyExists
			}
			return nil, fmt.Errorf("repository: update username: %w", err)
		}
	}

	if newPasswordHash != nil && currentUserType == "manual" {
		_, err = tx.Exec(ctx, `
			UPDATE manual_user SET password_hash = $1
			WHERE user_uuid = $2
		`, newPasswordHash, userUUID)
		if err != nil {
			return nil, fmt.Errorf("repository: update password: %w", err)
		}
	}

	var user model.User
	err = tx.QueryRow(ctx, `
		SELECT uuid, username, user_type, created_at, updated_at
		FROM base_user WHERE uuid = $1
	`, userUUID).Scan(
		&user.UUID, &user.Username, &user.UserType, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository: fetch updated user: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return &user, nil
}

// Logout - заглушка для stateless JWT архитектуры.
// Токен инвалидируется на клиенте (удаление из sessionStorage).
func (r *AuthRepository) Logout(ctx context.Context, userUUID uuid.UUID) error {
	// Stateless JWT: нет сессии на сервере для инвалидации
	// Если в будущем потребуется blacklist, этот метод можно доработать
	return nil
}
