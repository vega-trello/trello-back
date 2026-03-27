package dto

import "github.com/vega-trello/trello-back/internal/utils"

// post /projects/{projectUUID}/members
type CreateMemberRequest struct {
	UserUUID string `json:"userUUID"`
	RoleID   int    `json:"role_id" binding:"required,min=1"`
}

// patch /projects/{projectUUID}/member
type UpdateMemberRequest struct {
	RoleID int `json:"role_id" binding:"required,min=1"`
}

func (r *CreateMemberRequest) Validate() error {
	if r.UserUUID == "" {
		return &utils.ValidationError{
			Field:   "user_uuid",
			Message: "user_uuid is required",
		}
	}

	if r.RoleID < 1 {
		return &utils.ValidationError{
			Field:   "role_id",
			Message: "role_id must be at least 1",
		}
	}

	return nil
}

func (r *UpdateMemberRequest) Validate() error {
	if r.RoleID < 1 {
		return &utils.ValidationError{
			Field:   "role_id",
			Message: "role_id must be at least 1",
		}
	}

	return nil
}
