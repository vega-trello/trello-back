package dto

import (
	"errors"

	"github.com/vega-trello/trello-back/internal/utils"
)

// POST /api/v1/projects/:id/columns
type CreateColumnRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=64"`
	Position *int   `json:"position" binding:"omitempty,min=0"`
}

// PATCH /api/v1/columns/:id
type UpdateColumnRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=64"`
	Position *int   `json:"position,omitempty" binding:"omitempty,min=0"`
}

type MoveColumnRequest struct {
	Direction string `json:"direction" binding:"required,oneof=left right"`
}

func (r *MoveColumnRequest) Validate() error {
	if r.Direction != "left" && r.Direction != "right" {
		return errors.New("direction must be 'left' or 'right'")
	}
	return nil
}

func (r *CreateColumnRequest) Validate() error {
	if r.Name == "" || len(r.Name) > 64 {
		return &utils.ValidationError{
			Field:   "name",
			Message: "name must be between 1 and 64 characters",
		}
	}
	return nil
}

func (r *UpdateColumnRequest) Validate() error {
	if r.Name == "" || len(r.Name) > 64 {
		return &utils.ValidationError{
			Field:   "name",
			Message: "name must be between 1 and 64 characters",
		}
	}
	return nil
}
