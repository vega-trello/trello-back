package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/vega-trello/trello-back/internal/model"
)

// LoginResponse - POST /auth/login 200
// Возвращает только токен
type LoginResponse struct {
	Token string `json:"token"`
}

// RegisterResponse — POST /auth/register 201
type RegisterResponse struct {
	Token string `json:"token"`
}

// SSOExchangeResponse — POST /auth/sso/exchange 200
type SSOExchangeResponse struct {
	Token string `json:"token"`
}

// UserResponse — базовая информация о пользователе
type UserResponse struct {
	UUID     uuid.UUID `json:"uuid"`
	Username string    `json:"username"`
	UserType string    `json:"user_type"`
}

// SelfUserResponse — расширенная информация о текущем пользователе (GET /user)
type SelfUserResponse struct {
	UUID      uuid.UUID `json:"uuid"`
	Username  string    `json:"username"`
	UserType  string    `json:"user_type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FromSelfUserModel конвертирует model.SelfUser - SelfUserResponse
func FromSelfUserModel(u *model.SelfUser) SelfUserResponse {
	return SelfUserResponse{
		UUID:      u.UUID,
		Username:  u.Username,
		UserType:  u.UserType,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// FromUserModel конвертирует model.User - UserResponse
func FromUserModel(u *model.User) UserResponse {
	return UserResponse{
		UUID:     u.UUID,
		Username: u.Username,
	}
}

// UserInfo - для вложенных ответов
type UserInfo struct {
	UUID     uuid.UUID `json:"uuid"`
	Username string    `json:"username"`
	UserType string    `json:"user_type"`
}

// ErrorResponse - универсальный формат ошибки
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
