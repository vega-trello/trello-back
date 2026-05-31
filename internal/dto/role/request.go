package dto

import "errors"

type CreateRoleRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	PermissionIDs []int  `json:"permission_ids"`
}

func (r CreateRoleRequest) Validate() error {
	if r.Name == "" || len(r.Name) > 32 {
		return errors.New("name must be between 1 and 32 characters")
	}

	if r.Description != "" && len(r.Description) > 256 {
		return errors.New("description must not exceed 256 characters")
	}

	return nil
}

type UpdateRoleRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	PermissionIDs []int  `json:"permission_ids"`
}

func (r UpdateRoleRequest) Validate() error {
	if r.Name == "" || len(r.Name) > 32 {
		return errors.New("name must be between 1 and 32 characters")
	}

	if r.Description != "" && len(r.Description) > 256 {
		return errors.New("description must not exceed 256 characters")
	}

	return nil
}
