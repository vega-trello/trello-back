package dto

import "github.com/vega-trello/trello-back/internal/utils"

// POST /api/v1/projects/:id/members
type AddMemberRequest struct {
	Username string `json:"username" binding:"required"`
	RoleID   int    `json:"role_id" binding:"required,min=1"`
}

func (r *AddMemberRequest) Validate() error {
	if r.Username == "" {
		return &utils.ValidationError{Field: "username", Message: "username is required"}
	}
	if r.RoleID < 1 {
		return &utils.ValidationError{Field: "role_id", Message: "role_id must be at least 1"}
	}
	return nil
}

// PUT /api/v1/projects/:id/members/:userId
type UpdateMemberRoleRequest struct {
	RoleID int `json:"role_id" binding:"required,min=1"`
}

func (r *UpdateMemberRoleRequest) Validate() error {
	if r.RoleID < 1 {
		return &utils.ValidationError{Field: "role_id", Message: "role_id must be at least 1"}
	}
	return nil
}
