package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/vega-trello/trello-back/internal/model"
	"github.com/vega-trello/trello-back/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrUserAlreadyExists  = errors.New("username already taken") // 🔥 Добавлена локально
	ErrSSOProviderError   = errors.New("SSO provider returned an error")
	ErrInvalidSSOToken    = errors.New("invalid or expired SSO token")
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

	user, err := s.repo.RegisterPasswordUser(ctx, username, hash)
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("service: register: %w", err)
	}
	return user, nil
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

// LoginBySSO аутентифицирует пользователя через корпоративный SSO
func (s *UserService) LoginBySSO(
	ctx context.Context,
	provider string,
	externalID string,
	username string,
	metadata json.RawMessage,
) (*model.User, error) {
	// 🔹 Валидация входных параметров
	if provider == "" || externalID == "" || username == "" {
		return nil, fmt.Errorf("service: provider, externalID and username are required")
	}

	user, err := s.repo.FindOrCreateUserBySSO(ctx, provider, externalID, username, metadata)
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("service: SSO login: %w", err)
	}

	return user, nil
}

// GetProfile возвращает расширенную информацию о текущем пользователе
func (s *UserService) GetProfile(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error) {
	user, err := s.repo.GetSelfUser(ctx, userUUID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, repository.ErrUserNotFound
		}
		return nil, fmt.Errorf("service: get profile: %w", err)
	}
	return user, nil
}

// UpdateProfile обновляет имя и/или пароль текущего пользователя
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

	updated, err := s.repo.UpdateSelfUser(ctx, userUUID, oldPassword, newUsername, newPassword)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidCredentials) {
			return nil, ErrInvalidCredentials
		}
		if errors.Is(err, repository.ErrSSOUserPasswordChange) {
			return nil, repository.ErrSSOUserPasswordChange
		}
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return nil, ErrUserAlreadyExists
		}
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, repository.ErrUserNotFound
		}
		return nil, fmt.Errorf("service: update profile: %w", err)
	}
	return updated, nil
}

// Logout - заглушка для stateless JWT
func (s *UserService) Logout(ctx context.Context, userUUID uuid.UUID) error {
	return s.repo.Logout(ctx, userUUID)
}
