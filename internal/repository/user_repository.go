package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vega-trello/trello-back/internal/model"
	"golang.org/x/crypto/bcrypt"
)

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
		UserType: "manual",
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

	user.UserType = "manual"
	return &user, passwordHash, nil
}

// FindOrCreateUserBySSO находит или создаёт SSO-пользователя (JIT Provisioning)
func (r *UserRepository) FindOrCreateUserBySSO(
	ctx context.Context,
	provider string,
	externalID string,
	username string,
	metadata json.RawMessage,
) (*model.User, error) {
	var user model.User

	err := r.db.QueryRow(ctx, `
		SELECT u.uuid, u.username
		FROM base_user u
		JOIN sso_user s ON u.uuid = s.user_uuid
		WHERE s.provider = $1 AND s.external_id = $2
	`, provider, externalID).Scan(&user.UUID, &user.Username)

	if err == nil {
		user.UserType = "sso"
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

	metaToSave := metadata
	if metaToSave == nil {
		metaToSave = json.RawMessage("{}")
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO sso_user (user_uuid, provider, external_id, metadata)
		VALUES ($1, $2, $3, $4)
	`, newUUID, provider, externalID, metaToSave)
	if err != nil {
		return nil, fmt.Errorf("repository: create sso_user record: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return &model.User{
		UUID:     newUUID,
		Username: username,
		UserType: "sso",
	}, nil
}

// FindUserByUUID находит пользователя по UUID (для валидации JWT)
func (r *UserRepository) FindUserByUUID(
	ctx context.Context,
	userUUID uuid.UUID,
) (*model.User, error) {
	var user model.User

	err := r.db.QueryRow(ctx, `
		SELECT uuid, username, user_type
		FROM base_user WHERE uuid = $1
	`, userUUID).Scan(&user.UUID, &user.Username, &user.UserType)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find user by uuid: %w", err)
	}

	return &user, nil
}

// GetSelfUser возвращает расширенную информацию о пользователе для эндпоинта GET /self
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

// UpdateSelfUser обновляет профиль текущего пользователя для эндпоинта PATCH /self
func (r *UserRepository) UpdateSelfUser(
	ctx context.Context,
	userUUID uuid.UUID,
	newUsername *string,
	newPasswordHash *string,
) (*model.SelfUser, error) {
	if newUsername == nil && newPasswordHash == nil {
		return nil, errors.New("repository: no fields to update")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var userType string
	err = tx.QueryRow(ctx, `
		SELECT user_type FROM base_user WHERE uuid = $1
	`, userUUID).Scan(&userType)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find user: %w", err)
	}

	// SSO-пользователи не могут менять пароль
	if userType != "manual" && newPasswordHash != nil {
		return nil, ErrSSOUserPasswordChange
	}

	updates := make(map[string]interface{})
	if newUsername != nil {
		updates["username"] = *newUsername
	}
	if newPasswordHash != nil && userType == "manual" {
		updates["password_hash"] = *newPasswordHash
	}

	if len(updates) == 0 {
		return nil, errors.New("repository: no valid fields to update")
	}

	if newUsername != nil {
		_, err = tx.Exec(ctx, `
			UPDATE base_user 
			SET username = $1, updated_at = NOW()
			WHERE uuid = $2
		`, *newUsername, userUUID)
		if err != nil {
			if isUniqueViolation(err, "base_user_username_key") {
				return nil, ErrUserAlreadyExists
			}
			return nil, fmt.Errorf("repository: update username: %w", err)
		}
	}

	if newPasswordHash != nil && userType == "manual" {
		_, err = tx.Exec(ctx, `
			UPDATE manual_user 
			SET password_hash = $1
			WHERE user_uuid = $2
		`, *newPasswordHash, userUUID)
		if err != nil {
			return nil, fmt.Errorf("repository: update password: %w", err)
		}
	}

	if newUsername != nil || (newPasswordHash != nil && userType == "manual") {
		_, err = tx.Exec(ctx, `
			UPDATE base_user 
			SET updated_at = NOW()
			WHERE uuid = $1
		`, userUUID)
		if err != nil {
			return nil, fmt.Errorf("repository: update updated_at: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("repository: commit transaction: %w", err)
	}

	return r.GetSelfUser(ctx, userUUID)
}

// VerifyPassword проверяет пароль для manual-пользователя
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

// Logout — заглушка для stateless JWT (инвалидация на клиенте)
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

// isForeignKeyViolation проверяет, является ли ошибкой нарушение внешнего ключа
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}
	return false
}
