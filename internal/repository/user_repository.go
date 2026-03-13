package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vega-trello/trello-back/internal/dto/auth"
	"github.com/vega-trello/trello-back/internal/model"
)

type UserRepositoryInterface interface {
	CreateManualUser(ctx context.Context, username string, passwordHash []byte) (*dto.UserInfo, error)
	FindByUsername(ctx context.Context, username string) (*dto.UserInfo, []byte, error)
}
type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateManualUser(ctx context.Context, username string, passwordHash []byte) (*dto.UserInfo, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback(ctx)

	var user model.User

	err = tx.QueryRow(ctx, `
		INSERT INTO base_user (username, user_type, created_at, updated_at)
		VALUES ($1, 'manual', NOW(), NOW())
		RETURNING uuid, username, user_type, created_at, updated_at
	`, username).Scan(
		&user.UUID,
		&user.Username,
		&user.UserType,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO manual_user (user_uuid, password_hash)
		VALUES ($1, $2)
	`, user.UUID, passwordHash)

	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.UserInfo{
		UUID:      user.UUID.String(),
		Username:  user.Username,
		UserType:  user.UserType,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*dto.UserInfo, []byte, error) {
	var user model.User
	var passwordHash []byte

	err := r.db.QueryRow(ctx, `
		SELECT bu.uuid, bu.username, bu.user_type, bu.created_at, bu.updated_at, mu.password_hash
		FROM base_user bu
		LEFT JOIN manual_user mu ON bu.uuid = mu.user_uuid
		WHERE bu.username = $1
	`, username).Scan(
		&user.UUID,
		&user.Username,
		&user.UserType,
		&user.CreatedAt,
		&user.UpdatedAt,
		&passwordHash,
	)

	if err != nil {
		return nil, nil, err
	}

	return &dto.UserInfo{
		UUID:      user.UUID.String(),
		Username:  user.Username,
		UserType:  user.UserType,
		CreatedAt: user.CreatedAt,
	}, passwordHash, nil
}

var _ UserRepositoryInterface = (*UserRepository)(nil)
