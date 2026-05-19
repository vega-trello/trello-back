package dto

import "github.com/vega-trello/trello-back/internal/utils"

// RegisterRequest - POST /auth/register
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=1,max=32"`
	Password string `json:"password" binding:"required,min=8"`
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

// LoginRequest - POST /auth/login
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=1,max=32"`
	Password string `json:"password" binding:"required"`
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

// SSOExchangeRequest - POST /auth/sso/exchange
type SSOExchangeRequest struct {
	Token string `json:"token" binding:"required"`
}

func (r *SSOExchangeRequest) Validate() error {
	if r.Token == "" {
		return &utils.ValidationError{
			Field:   "token",
			Message: "SSO token is required",
		}
	}
	return nil
}

// LogoutRequest - POST /auth/logout (stateless JWT)
type LogoutRequest struct{}

func (r *LogoutRequest) Validate() error {
	return nil
}

// UpdateUserRequest - PATCH /user
type UpdateUserRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	Username    string `json:"username" binding:"omitempty,min=1,max=32"`
	Password    string `json:"password" binding:"omitempty,min=8"`
}

func (r *UpdateUserRequest) Validate() error {
	if r.OldPassword == "" {
		return &utils.ValidationError{Field: "old_password", Message: "old_password is required"}
	}

	if r.Username != "" {
		if len(r.Username) < 1 || len(r.Username) > 32 {
			return &utils.ValidationError{
				Field:   "username",
				Message: "username must be between 1 and 32 characters",
			}
		}
		if !utils.IsValidUsername(r.Username) {
			return &utils.ValidationError{
				Field:   "username",
				Message: "username can contain only lowercase letters, numbers, underscore and dash",
			}
		}
	}

	if r.Password != "" && len(r.Password) < 8 {
		return &utils.ValidationError{
			Field:   "password",
			Message: "password must be at least 8 characters",
		}
	}
	return nil
}
