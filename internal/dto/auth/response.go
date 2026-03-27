package dto

import "time"

// post: /auth/login 200
type LoginResponse struct {
	Token string `json:"token"`
}

// post: /auth/update 200
type UpdateTokenResponse struct {
	Token string `json:"token"`
}

// get: /user 200
type UserResponse struct {
	Username string `json:"username"`
	UUID     string `json:"uuid"`
}

// get:: /user 200 (self user)
type SelfUserResponse struct {
	UserResponse
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	UserType  string    `json:"userType"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
type UserInfo struct {
	UUID      string    `json:"uuid"`
	Username  string    `json:"username"`
	UserType  string    `json:"user_type"`
	CreatedAt time.Time `json:"created_at"`
}
