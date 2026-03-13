package service

import (
	"context"
	"errors"

	dto "github.com/vega-trello/trello-back/internal/dto/auth"
	"github.com/vega-trello/trello-back/internal/repository"
	"github.com/vega-trello/trello-back/internal/utils"
)

type AuthServiceInterface interface {
	Register(ctx context.Context, username, password string) (*dto.AuthResponse, error)
	Login(ctx context.Context, username, password string) (*dto.AuthResponse, error)
}

type AuthService struct {
	userRepo repository.UserRepositoryInterface
}

func NewAuthService(userRepo repository.UserRepositoryInterface) AuthServiceInterface {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) Register(ctx context.Context, username, password string) (*dto.AuthResponse, error) {
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	userInfo, err := s.userRepo.CreateManualUser(ctx, username, passwordHash)
	if err != nil {
		return nil, err
	}

	// Пока это мок: "dev-" + UUID
	// В продакшене будет JWT с подписью и expiry
	token := "dev-" + userInfo.UUID

	return &dto.AuthResponse{
		Token: token,
		User:  *userInfo,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*dto.AuthResponse, error) {
	userInfo, passwordHash, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	err = utils.ComparePassword(passwordHash, password)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	token := "dev-" + userInfo.UUID

	return &dto.AuthResponse{
		Token: token,
		User:  *userInfo,
	}, nil
}

var _ AuthServiceInterface = (*AuthService)(nil)
