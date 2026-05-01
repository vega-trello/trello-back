// internal/dto/role/request.go
package dto // ← Обязательно 'dto', а не 'role'!

import "github.com/vega-trello/trello-back/internal/utils"

type CreateRoleRequest struct {
	Name          string  `json:"name" binding:"required,min=1,max=32"`
	Description   *string `json:"description,omitempty" binding:"omitempty,max=256"`
	PermissionIDs *[]int  `json:"permission_ids,omitempty"`
}

type UpdateRoleRequest struct {
	Name          *string `json:"name,omitempty" binding:"omitempty,min=1,max=32"`
	Description   *string `json:"description,omitempty" binding:"omitempty,max=256"`
	PermissionIDs *[]int  `json:"permission_ids,omitempty"`
}

func (r *CreateRoleRequest) Validate() error {
	if r.Name == "" {
		return &utils.ValidationError{Field: "name", Message: "required"}
	}
	return nil
}

func (r *UpdateRoleRequest) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return &utils.ValidationError{Field: "name", Message: "cannot be empty"}
	}
	return nil
}
