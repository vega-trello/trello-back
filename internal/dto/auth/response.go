package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/vega-trello/trello-back/internal/model"
)

// POST /auth/login 200
type LoginResponse struct {
	Token string `json:"token"`
}

// POST /auth/refresh 200
type UpdateTokenResponse struct {
	Token string `json:"token"`
}

// GET /user 200 — базовая информация о пользователе
type UserResponse struct {
	UUID     uuid.UUID `json:"uuid"`
	Username string    `json:"username"`
}

// GET /user 200 (self) — расширенная информация о текущем пользователе
type SelfUserResponse struct {
	UUID      uuid.UUID `json:"uuid"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserType  string    `json:"user_type"`
}

// FromSelfUserModel конвертирует model.SelfUser - SelfUserResponse
func FromSelfUserModel(u *model.SelfUser) SelfUserResponse {
	return SelfUserResponse{
		UUID:      u.UUID,
		Username:  u.Username,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		UserType:  u.UserType,
	}
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// UserInfo - вспомогательная структура (например, для вложенных ответов)
type UserInfo struct {
	UUID      uuid.UUID `json:"uuid"`
	Username  string    `json:"username"`
	UserType  string    `json:"user_type"`
	CreatedAt time.Time `json:"created_at"`
}
