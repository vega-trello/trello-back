package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vega-trello/trello-back/internal/model"
)

type UserRepositoryInterface interface {
	FindByID(ctx context.Context, userUUID uuid.UUID) (*model.User, error)
	Update(ctx context.Context, userUUID uuid.UUID, newUsername *string, newPasswordHash []byte) (*model.User, error)
}

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// FindByID возвращает пользователя по UUID
// GET /user (после извлечения UUID из JWT)
func (r *UserRepository) FindByID(
	ctx context.Context,
	userUUID uuid.UUID,
) (*model.User, error) {
	var user model.User

	err := r.db.QueryRow(ctx, `
		SELECT uuid, username, user_type, created_at, updated_at
		FROM base_user
		WHERE uuid = $1
	`, userUUID).Scan(
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
		return nil, fmt.Errorf("repository: find user by id: %w", err)
	}

	return &user, nil
}

// Update обновляет username и/или password пользователя
// newUsername: nil = не менять, "" = ошибка (пустой не разрешён), строка = новое значение
// newPasswordHash: nil/пустой = не менять, иначе = новый хеш пароля
// Обновляет updated_at в base_user в любом случае
// PATCH /user
func (r *UserRepository) Update(
	ctx context.Context,
	userUUID uuid.UUID,
	newUsername *string,
	newPasswordHash []byte,
) (*model.User, error) {
	if (newUsername == nil || *newUsername == "") && len(newPasswordHash) == 0 {
		return r.FindByID(ctx, userUUID)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Обновляем username, если передан
	if newUsername != nil && *newUsername != "" {
		_, err = tx.Exec(ctx, `
			UPDATE base_user
			SET username = $1, updated_at = NOW()
			WHERE uuid = $2
		`, *newUsername, userUUID)

		if err != nil {
			if IsUniqueViolation(err) {
				return nil, ErrUsernameTaken
			}
			return nil, fmt.Errorf("repository: update username: %w", err)
		}
	}

	// Обновляем password, если передан
	// Обновляем только для manual-пользователей (у SSO пароля нет)
	if len(newPasswordHash) > 0 {
		result, err := tx.Exec(ctx, `
			UPDATE manual_user
			SET password_hash = $1
			WHERE user_uuid = $2
		`, newPasswordHash, userUUID)

		if err != nil {
			return nil, fmt.Errorf("repository: update password: %w", err)
		}

		// Проверяем, что пользователь существует и имеет manual тип
		if result.RowsAffected() == 0 {
			// Может быть SSO пользователь или не существует
			// Проверяем существование в base_user
			var userType string
			err := tx.QueryRow(ctx, `SELECT user_type FROM base_user WHERE uuid = $1`, userUUID).Scan(&userType)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrUserNotFound
			}
			if userType != "manual" {
				return nil, fmt.Errorf("repository: cannot update password for %s user", userType)
			}
			return nil, fmt.Errorf("repository: manual_user record not found for user %s", userUUID)
		}

		// Если username не обновляли — обновим updated_at вручную
		if newUsername == nil || *newUsername == "" {
			_, err = tx.Exec(ctx, `
				UPDATE base_user
				SET updated_at = NOW()
				WHERE uuid = $1
			`, userUUID)
			if err != nil {
				return nil, fmt.Errorf("repository: update updated_at: %w", err)
			}
		}
	}

	var user model.User
	err = tx.QueryRow(ctx, `
		SELECT uuid, username, user_type, created_at, updated_at
		FROM base_user
		WHERE uuid = $1
	`, userUUID).Scan(
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
		return nil, fmt.Errorf("repository: fetch updated user: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return &user, nil
}
