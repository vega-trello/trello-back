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
	ErrInvalidCredentials    = errors.New("invalid username or password")
	ErrPasswordTooShort      = errors.New("password must be at least 8 characters")
	ErrUserAlreadyExists     = errors.New("username already taken")
	ErrSSOProviderError      = errors.New("SSO provider returned an error")
	ErrInvalidSSOToken       = errors.New("invalid or expired SSO token")
	ErrSSOUserPasswordChange = errors.New("SSO users cannot change password via this endpoint")
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

func (s *UserService) LoginBySSO(
	ctx context.Context,
	provider string,
	externalID string,
	username string,
	metadata json.RawMessage,
) (*model.User, error) {
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

func (s *UserService) GetOtherUserProfile(ctx context.Context, targetUserUUID uuid.UUID) (*model.User, error) {
	user, err := s.repo.FindUserByUUID(ctx, targetUserUUID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, fmt.Errorf("service: user not found: %w", repository.ErrUserNotFound)
		}
		return nil, fmt.Errorf("service: get other user profile: %w", err)
	}

	return &model.User{
		UUID:     user.UUID,
		Username: user.Username,
		UserType: user.UserType,
	}, nil
}

func (s *UserService) GetSelfProfile(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error) {
	user, err := s.repo.GetSelfUser(ctx, userUUID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, fmt.Errorf("service: get profile: %w", repository.ErrUserNotFound)
		}
		return nil, fmt.Errorf("service: get profile: %w", err)
	}
	return user, nil
}

func (s *UserService) UpdateSelfProfile(
	ctx context.Context,
	userUUID uuid.UUID,
	newUsername *string,
	newPassword *string,
) (*model.SelfUser, error) {
	if newUsername == nil && newPassword == nil {
		return nil, fmt.Errorf("service: at least one field (username or password) must be provided")
	}

	var hashedPassword *string
	if newPassword != nil {
		if len(*newPassword) < 8 {
			return nil, ErrPasswordTooShort
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*newPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("service: hash password: %w", err)
		}
		hashStr := string(hash)
		hashedPassword = &hashStr
	}

	updated, err := s.repo.UpdateSelfUser(ctx, userUUID, newUsername, hashedPassword)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidCredentials) {
			return nil, ErrInvalidCredentials
		}
		if errors.Is(err, repository.ErrSSOUserPasswordChange) {
			return nil, ErrSSOUserPasswordChange
		}
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return nil, ErrUserAlreadyExists
		}
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, fmt.Errorf("service: update profile: %w", repository.ErrUserNotFound)
		}
		return nil, fmt.Errorf("service: update profile: %w", err)
	}

	return updated, nil
}

// Logout - заглушка для stateless JWT
func (s *UserService) Logout(ctx context.Context, userUUID uuid.UUID) error {
	// Stateless JWT: нет сессии на сервере для инвалидации
	// В будущем здесь можно добавить:
	// - логирование выхода для аудита
	// - добавление токена в блэклист (Redis) для принудительной инвалидации
	return s.repo.Logout(ctx, userUUID)
}
