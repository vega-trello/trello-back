package dto

import "github.com/vega-trello/trello-back/internal/utils"

// POST /projects/{uuid}/statuses
type CreateStatusRequest struct {
	Name string `json:"name" binding:"required,min=1,max=32"`
}

func (r *CreateStatusRequest) Validate() error {
	if r.Name == "" {
		return &utils.ValidationError{Field: "name", Message: "name is required"}
	}
	return nil
}

// UpdateStatusRequest — PATCH /statuses/{id}
// Используем omitempty для удобства клиента (частичное обновляем)
type UpdateStatusRequest struct {
	Name string `json:"name" binding:"required,min=1,max=32"`
}

func (r *UpdateStatusRequest) Validate() error {
	if r.Name == "" || len(r.Name) > 32 {
		return &utils.ValidationError{
			Field:   "name",
			Message: "name must be between 1 and 32 characters",
		}
	}
	return nil
}
