package dto

import (
	"github.com/vega-trello/trello-back/internal/utils"
)

// POST /api/v1/projects/:id/statuses
type CreateStatusRequest struct {
	Name string `json:"name" binding:"required,min=1,max=32"`
}

func (r *CreateStatusRequest) Validate() error {
	if r.Name == "" {
		return &utils.ValidationError{Field: "name", Message: "name is required"}
	}
	return nil
}

// PUT /api/v1/statuses/:id
type UpdateStatusRequest struct {
	Name string `json:"name" binding:"omitempty,min=1,max=32"`
}
