package repository

import (
	"errors"
)

var (
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUsernameTaken       = errors.New("username already taken")
	ErrProjectNotFound     = errors.New("project not found")
	ErrMemberNotFound      = errors.New("member not found")
	ErrMemberAlreadyExists = errors.New("member already exists in project")
	ErrColumnNotFound      = errors.New("column not found")
)
