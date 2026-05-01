package repository

import (
	"errors"
)

var (
	ErrUserAlreadyExists      = errors.New("user already exists")
	ErrUserNotFound           = errors.New("user not found")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrUsernameTaken          = errors.New("username already taken")
	ErrProjectNotFound        = errors.New("project not found")
	ErrMemberNotFound         = errors.New("member not found")
	ErrMemberAlreadyExists    = errors.New("member already exists in project")
	ErrColumnNotFound         = errors.New("column not found")
	ErrTaskNotFound           = errors.New("task not found")
	ErrTaskDeleted            = errors.New("task is deleted")
	ErrTaskArchived           = errors.New("task is archived")
	ErrAssigneeNotFound       = errors.New("assignee not found")
	ErrAssigneeAlreadyExists  = errors.New("user already assigned to this task")
	ErrTaskNotFoundAssgnee    = errors.New("task not found or does not belong to project")
	ErrRoleNotFound           = errors.New("role not found")
	ErrRoleAlreadyExists      = errors.New("role with this name already exists in project")
	ErrCannotDeleteSystemRole = errors.New("cannot delete system role")
	ErrPermissionNotFound     = errors.New("permission not found")
	ErrSSOUserPasswordChange  = errors.New("cannot change password for SSO user")
	ErrAccessDenied           = errors.New("user does not have access to this project")
	ErrInvalidColumn          = errors.New("column does not belong to the specified project")
	ErrInvalidPosition        = errors.New("position index out of bounds")
	ErrTagNotFound            = errors.New("tag not found")
	ErrAlreadyAssigned        = errors.New("user is already assigned to this task")
)
