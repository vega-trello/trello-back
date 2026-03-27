package dto

import (
	"github.com/vega-trello/trello-back/internal/utils"
)

type CreateRoleRequest struct {
	Name          string `json:"name" binding:"required,min=1,max=32"`
	Description   string `json:"description,omitempty" binding:"omitempty,max=256"`
	ProjectUUID   string `json:"project_uuid" binding:"required"`
	PermissionIDs []int  `json:"permission_ids,omitempty"`
}

type UpdateRoleRequest struct {
	Name          string `json:"name" binding:"required,min=1,max=32"`
	Description   string `json:"description,omitempty" binding:"omitempty,max=256"`
	PermissionIDs []int  `json:"permission_ids,omitempty"`
}

func (r *CreateRoleRequest) Validate() error {
	if r.Name == "" {
		return &utils.ValidationError{
			Field:   "name",
			Message: "name is required",
		}
	}

	if r.ProjectUUID == "" {
		return &utils.ValidationError{
			Field:   "project_uuid",
			Message: "project_uuid is required",
		}
	}

	return nil
}

func (r *UpdateRoleRequest) Validate() error {
	if r.Name != "" && len(r.Name) > 32 {
		return &utils.ValidationError{
			Field:   "name",
			Message: "name must be at most 32 characters",
		}
	}
	
	return nil
}
