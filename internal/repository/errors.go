package repository

import (
	"errors"
)

var (
	ErrUserAlreadyExists     = errors.New("user already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrUsernameTaken         = errors.New("username already taken")
	ErrProjectNotFound       = errors.New("project not found")
	ErrMemberNotFound        = errors.New("member not found")
	ErrMemberAlreadyExists   = errors.New("member already exists in project")
	ErrColumnNotFound        = errors.New("column not found")
	ErrTaskNotFound          = errors.New("task not found")
	ErrTaskDeleted           = errors.New("task is deleted")
	ErrTaskArchived          = errors.New("task is archived")
	ErrAssigneeNotFound      = errors.New("assignee not found")
	ErrAssigneeAlreadyExists = errors.New("user already assigned to this task")
	ErrTaskNotFoundAssgnee   = errors.New("task not found or does not belong to project")
)
