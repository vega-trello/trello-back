package dto

import "github.com/vega-trello/trello-back/internal/utils"

// post: /auth/register
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=8"`
}

// post: /auth/login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// post: /auth/update
type UpdateTokenRequest struct {
	RefreshToken string `json:"token" binding:"required"`
}

// post: /auth/logout
type LogoutRequest struct{}

func (r *LogoutRequest) Validate() error {
	return nil
}

func (r *RegisterRequest) Validate() error {
	if r.Username == "" {
		return &utils.ValidationError{Field: "username", Message: "username is required"}
	}

	if !utils.IsValidUsername(r.Username) {
		return &utils.ValidationError{
			Field:   "username",
			Message: "username can contain only lowercase letters, numbers, underscore and dash",
		}
	}

	if len(r.Password) < 8 {
		return &utils.ValidationError{
			Field:   "password",
			Message: "password must be at least 8 characters",
		}
	}

	return nil
}

func (r *LoginRequest) Validate() error {
	if r.Username == "" {
		return &utils.ValidationError{Field: "username", Message: "username is required"}
	}

	if r.Password == "" {
		return &utils.ValidationError{Field: "password", Message: "password is required"}
	}

	return nil
}

func (r *UpdateTokenRequest) Validate() error {
	if r.RefreshToken == "" {
		return &utils.ValidationError{
			Field:   "refreshToken",
			Message: "refreshToken is required",
		}
	}
	return nil
}
