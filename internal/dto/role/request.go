package dto

import "errors"

type CreateRoleRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	PermissionIDs []int  `json:"permission_ids"`
}

func (r CreateRoleRequest) Validate() error {
	// Name: required, minLength 1, maxLength 32
	if r.Name == "" || len(r.Name) > 32 {
		return errors.New("name must be between 1 and 32 characters")
	}
	// Description: optional, maxLength 256
	if r.Description != "" && len(r.Description) > 256 {
		return errors.New("description must not exceed 256 characters")
	}
	// PermissionIDs: required, non-empty
	if len(r.PermissionIDs) == 0 {
		return errors.New("at least one permission_id is required")
	}
	return nil
}

type UpdateRoleRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	PermissionIDs []int  `json:"permission_ids"`
}

func (r UpdateRoleRequest) Validate() error {
	// Name: required, 1..32
	if r.Name == "" || len(r.Name) > 32 {
		return errors.New("name must be between 1 and 32 characters")
	}
	// Description: optional, maxLength 256
	if r.Description != "" && len(r.Description) > 256 {
		return errors.New("description must not exceed 256 characters")
	}
	// PermissionIDs: required, non-empty
	if len(r.PermissionIDs) == 0 {
		return errors.New("at least one permission_id is required")
	}
	return nil
}
