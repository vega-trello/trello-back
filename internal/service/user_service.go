package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/vega-trello/trello-back/internal/model"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
)

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

// Register создаёт нового manual-пользователя
func (s *UserService) Register(ctx context.Context, username string, password string) (*model.User, error) {
	if len(password) < 8 {
		return nil, ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("service: hash password: %w", err)
	}

	return s.repo.RegisterPasswordUser(ctx, username, hash)
}

// Login аутентифицирует пользователя по логину и паролю
func (s *UserService) Login(ctx context.Context, username string, password string) (*model.User, error) {
	user, storedHash, err := s.repo.FindUserByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword(storedHash, []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// GetProfile возвращает расширенную информацию о текущем пользователе (GET /user)
func (s *UserService) GetProfile(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error) {
	return s.repo.GetSelfUser(ctx, userUUID)
}

// UpdateProfile обновляет имя и/или пароль (PATCH /user)
// oldPassword обязателен для manual-пользователей, проверяется на уровне репозитория
func (s *UserService) UpdateProfile(
	ctx context.Context,
	userUUID uuid.UUID,
	oldPassword string,
	newUsername string,
	newPassword string,
) (*model.SelfUser, error) {
	if newPassword != "" && len(newPassword) < 8 {
		return nil, ErrPasswordTooShort
	}

	return s.repo.UpdateSelfUser(ctx, userUUID, oldPassword, newUsername, newPassword)
}

// Logout инвалидирует сессию (для stateless JWT это заглушка)
func (s *UserService) Logout(ctx context.Context, userUUID uuid.UUID) error {
	return s.repo.Logout(ctx, userUUID)
}
