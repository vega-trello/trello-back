package dto

import (
	"github.com/vega-trello/trello-back/internal/utils"
	"regexp"
)

// CreateTaskRequest - POST /projects/{projectUUID}/tasks
type CreateTaskRequest struct {
	Title       string  `json:"title" binding:"omitempty,max=256"`
	Description string  `json:"description" binding:"omitempty,max=2048"`
	StartDate   *string `json:"start_date" binding:"omitempty"`
	EndDate     *string `json:"end_date" binding:"omitempty"`
	ColumnID    *int    `json:"column_id" binding:"omitempty"`
	StatusID    *int    `json:"status_id" binding:"omitempty"`
}

func (r *CreateTaskRequest) Validate() error {
	if len(r.Title) > 256 {
		return &utils.ValidationError{Field: "title", Message: "title must not exceed 256 characters"}
	}
	if r.Description != "" && len(r.Description) > 2048 {
		return &utils.ValidationError{Field: "description", Message: "description must not exceed 2048 characters"}
	}
	if r.ColumnID != nil && *r.ColumnID <= 0 {
		return &utils.ValidationError{Field: "column_id", Message: "column_id must be a positive integer"}
	}
	return nil
}

// UpdateTaskRequest - PATCH /projects/{projectUUID}/task?taskID={id}
type UpdateTaskRequest struct {
	Title       *string `json:"title" binding:"omitempty,max=256"`
	Description *string `json:"description" binding:"omitempty,max=2048"`
	StartDate   *string `json:"start_date,omitempty" binding:"omitempty"`
	EndDate     *string `json:"end_date,omitempty" binding:"omitempty"`
	ColumnID    *int    `json:"column_id" binding:"required"`
	Color       *string `json:"color" binding:"omitempty"`
	Done        *bool   `json:"done" binding:"required"`
	Archived    *bool   `json:"archived" binding:"required"`

	StatusID *int `json:"status_id,omitempty" binding:"omitempty"`
}

func (r *UpdateTaskRequest) Validate() error {
	if r.Title != nil && len(*r.Title) > 256 {
		return &utils.ValidationError{Field: "title", Message: "title must not exceed 256 characters"}
	}
	if r.Description != nil && len(*r.Description) > 2048 {
		return &utils.ValidationError{Field: "description", Message: "description must not exceed 2048 characters"}
	}

	if r.ColumnID == nil {
		return &utils.ValidationError{Field: "column_id", Message: "column_id is required"}
	}
	if *r.ColumnID <= 0 {
		return &utils.ValidationError{Field: "column_id", Message: "column_id must be a positive integer"}
	}

	if r.Color != nil && !isValidHexColor(*r.Color) {
		return &utils.ValidationError{Field: "color", Message: "color must be a valid HEX string (#RRGGBB)"}
	}

	if r.Done == nil {
		return &utils.ValidationError{Field: "done", Message: "done is required"}
	}
	if r.Archived == nil {
		return &utils.ValidationError{Field: "archived", Message: "archived is required"}
	}

	return nil
}

func isValidHexColor(s string) bool {
	matched, _ := regexp.MatchString(`^#[0-9A-Fa-f]{6}$`, s)
	return matched
}
