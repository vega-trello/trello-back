package dto

import (
	"github.com/vega-trello/trello-back/internal/utils"
)

// POST /api/v1/columns/:id/tasks
type CreateTaskRequest struct {
	Title       *string `json:"title" binding:"omitempty,max=256"`
	Description *string `json:"description" binding:"omitempty,max=4096"`
	Position    *int    `json:"position,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
}

// PUT /api/v1/tasks/:id
type UpdateTaskRequest struct {
	Title       *string `json:"title" binding:"omitempty,max=256"`
	Description *string `json:"description" binding:"omitempty,max=4096"`
	Position    *int    `json:"position,omitempty"`
	StatusID    *int    `json:"status_id,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
}

// PUT /api/v1/tasks/:id/move
type MoveTaskRequest struct {
	ColumnID int `json:"column_id" binding:"required"`
	Position int `json:"position" binding:"required,min=0"`
}

func (r *MoveTaskRequest) Validate() error {
	if r.ColumnID < 1 {
		return &utils.ValidationError{Field: "column_id", Message: "column_id is required"}
	}
	if r.Position < 0 {
		return &utils.ValidationError{Field: "position", Message: "position must be non-negative"}
	}
	return nil
}

// POST /api/v1/tasks/:id/assignees
type AssignTaskRequest struct {
	Username string `json:"username" binding:"required"`
}

func (r *AssignTaskRequest) Validate() error {
	if r.Username == "" {
		return &utils.ValidationError{Field: "username", Message: "username is required"}
	}
	return nil
}

// POST /api/v1/tasks/:id/archive
type ArchiveTaskRequest struct{}

// DELETE /api/v1/tasks/:id
type DeleteTaskRequest struct{}
