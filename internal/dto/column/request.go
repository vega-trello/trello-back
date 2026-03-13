package dto

import (
	"github.com/vega-trello/trello-back/internal/utils"
)

// POST /api/v1/projects/:id/columns
type CreateColumnRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=64"`
	Position *int   `json:"position,omitempty"`
}

func (r *CreateColumnRequest) Validate() error {
	if r.Name == "" {
		return &utils.ValidationError{Field: "name", Message: "name is required"}
	}
	return nil
}

// PUT /api/v1/columns/:id
type UpdateColumnRequest struct {
	Name     string `json:"name" binding:"omitempty,min=1,max=64"`
	Position *int   `json:"position,omitempty"`
}

func (r *UpdateColumnRequest) Validate() error {
	if r.Name != "" && len(r.Name) > 64 {
		return &utils.ValidationError{Field: "name", Message: "name must be at most 64 characters"}
	}
	return nil
}
