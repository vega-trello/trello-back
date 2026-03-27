package dto

import (
	"github.com/vega-trello/trello-back/internal/utils"
)

// POST /api/v1/projects/:id/columns
type CreateColumnRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=64"`
	Position *int   `json:"position" binding:"required,min=1,max=64"`
}

// pacth /api/v1/columns/:id
type UpdateColumnRequest struct {
	Name     string `json:"name" binding:"omitempty,min=1,max=64"`
	Position *int   `json:"position,omitempty"`
}

// patch /api/v1/colums/{columnID}/move
type MoveColumnRequest struct {
	Position int `json:"position" binding:"omitempty,min=1,max=64"`
}

func (r *MoveColumnRequest) Validate() error {
	if r.Position < 0 {
		return &utils.ValidationError{
			Field:   "position",
			Message: "Position must be greater than or equal to 0",
		}
	}
	return nil
}

func (r *CreateColumnRequest) Validate() error {
	if r.Name == "" {
		return &utils.ValidationError{Field: "name", Message: "name is required"}
	}
	return nil
}

func (r *UpdateColumnRequest) Validate() error {
	if r.Name != "" && len(r.Name) > 64 {
		return &utils.ValidationError{Field: "name", Message: "name must be at most 64 characters"}
	}
	return nil
}
