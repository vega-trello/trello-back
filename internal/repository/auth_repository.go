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

// используется для возврата данных при логине
// Включает password_hash для проверки в сервисе
type loginResult struct {
	User         model.User
	PasswordHash []byte
}

type AuthRepositoryInterface interface {
	Register(ctx context.Context, username string, passwordHash []byte) (*model.User, error)
	LoginByUsername(ctx context.Context, username string) (*model.User, []byte, error)
	UpdatePassword(ctx context.Context, userUUID uuid.UUID, newPasswordHash []byte) error
	Logout(ctx context.Context, userUUID uuid.UUID) error
	FindByUsername(ctx context.Context, username string) (*model.User, error)
}

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) Register(
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
	//делаем insert в base_user
	var user model.User
	err = tx.QueryRow(ctx, `
		INSERT INTO base_user (uuid, username, user_type, created_at, updated_at)
		VALUES ($1, $2, 'manual', $3, $3)
		RETURNING uuid, username, user_type, created_at, updated_at
	`, userUUID, username, now).Scan(
		&user.UUID,
		&user.Username,
		&user.UserType,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("repository: create base_user: %w", err)
	}

	//делаем insert в manual_user
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

	return &user, nil
}

func (r *AuthRepository) LoginByUsername(
	ctx context.Context,
	username string,
) (*model.User, []byte, error) {
	var result loginResult

	err := r.db.QueryRow(ctx, `
		SELECT u.uuid, u.username, u.user_type, u.created_at, u.updated_at, m.password_hash
		FROM base_user u
		JOIN manual_user m ON u.uuid = m.user_uuid
		WHERE u.username = $1 AND u.user_type = 'manual'
	`, username).Scan(
		&result.User.UUID,
		&result.User.Username,
		&result.User.UserType,
		&result.User.CreatedAt,
		&result.User.UpdatedAt,
		&result.PasswordHash,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, nil, fmt.Errorf("repository: find user by username: %w", err)
	}

	return &result.User, result.PasswordHash, nil
}

// UpdatePassword обновляет пароль пользователя
// POST /auth/update
func (r *AuthRepository) UpdatePassword(
	ctx context.Context,
	userUUID uuid.UUID,
	newPasswordHash []byte,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	//Обновляем password_hash в manual_user
	_, err = tx.Exec(ctx, `
		UPDATE manual_user
		SET password_hash = $1
		WHERE user_uuid = $2
	`, newPasswordHash, userUUID)
	if err != nil {
		return fmt.Errorf("repository: update password_hash: %w", err)
	}

	//Обновляем updated_at в base_user
	result, err := tx.Exec(ctx, `
		UPDATE base_user
		SET updated_at = NOW()
		WHERE uuid = $1
	`, userUUID)
	if err != nil {
		return fmt.Errorf("repository: update updated_at: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("repository: commit transaction: %w", err)
	}

	return nil
}

// Logout заглушка для stateless JWT
// POST /auth/logout
// В stateless JWT logout не требует действий на стороне сервера
func (r *AuthRepository) Logout(ctx context.Context, userUUID uuid.UUID) error {
	// Stateless JWT: токен просто удаляется на клиенте
	return nil
}

// FindByUsername без password_hash. Используется для проверки существования пользователя
func (r *AuthRepository) FindByUsername(
	ctx context.Context,
	username string,
) (*model.User, error) {
	var user model.User

	err := r.db.QueryRow(ctx, `
		SELECT uuid, username, user_type, created_at, updated_at
		FROM base_user
		WHERE username = $1
	`, username).Scan(
		&user.UUID,
		&user.Username,
		&user.UserType,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find user by username: %w", err)
	}

	return &user, nil
}
