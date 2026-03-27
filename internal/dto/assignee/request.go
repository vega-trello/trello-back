package dto

import (
	"github.com/vega-trello/trello-back/internal/utils"
)

type CreateAssigneeRequest struct {
	UserUUID string `json:"userUUID" binding:"required"`
}

func (r *CreateAssigneeRequest) Validate() error {
	if r.UserUUID == "" {
		return &utils.ValidationError{
			Field:   "user_uuid",
			Message: "user_uuid is required",
		}
	}

	return nil
}
