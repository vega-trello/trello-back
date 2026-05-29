package dto

import (
	"github.com/vega-trello/trello-back/internal/utils"
)

// CreateTaskRequest - POST /projects/{projectUUID}/tasks
type CreateTaskRequest struct {
	Title       string  `json:"title" binding:"required,min=1,max=256"`
	Description string  `json:"description" binding:"omitempty,max=2048"`
	StartDate   *string `json:"start_date" binding:"omitempty"`
	EndDate     *string `json:"end_date" binding:"omitempty"`
	ColumnID    *int    `json:"column_id" binding:"required"` // 🔹 При создании тоже лучше требовать
	StatusID    *int    `json:"status_id" binding:"omitempty"`
}

func (r *CreateTaskRequest) Validate() error {
	if r.Title == "" || len(r.Title) > 256 {
		return &utils.ValidationError{Field: "title", Message: "title must be between 1 and 256 characters"}
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
	Title       *string `json:"title" binding:"omitempty,min=1,max=256"`
	Description *string `json:"description" binding:"omitempty,max=2048"`
	StartDate   *string `json:"start_date,omitempty" binding:"omitempty"`
	EndDate     *string `json:"end_date,omitempty" binding:"omitempty"`

	ColumnID *int `json:"column_id" binding:"required"`

	StatusID *int  `json:"status_id,omitempty" binding:"omitempty"`
	Archived *bool `json:"archived" binding:"omitempty"`
}

func (r *UpdateTaskRequest) Validate() error {
	if r.Title != nil && (len(*r.Title) == 0 || len(*r.Title) > 256) {
		return &utils.ValidationError{Field: "title", Message: "title must be between 1 and 256 characters"}
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

	return nil
}
