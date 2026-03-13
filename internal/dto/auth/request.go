package dto

import "github.com/vega-trello/trello-back/internal/utils"

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
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
