package dto

import (
	"github.com/vega-trello/trello-back/internal/utils"
)

// POST /api/v1/projects/:id/tags
type CreateTagRequest struct {
	Name  string `json:"name" binding:"required,min=1,max=64"`
	Color int    `json:"color" binding:"required"`
}

func (r *CreateTagRequest) Validate() error {
	if r.Name == "" {
		return &utils.ValidationError{Field: "name", Message: "name is required"}
	}
	return nil
}

// PUT /api/v1/tags/:id
type UpdateTagRequest struct {
	Name  string `json:"name" binding:"omitempty,min=1,max=64"`
	Color *int   `json:"color,omitempty"`
}

// POST /api/v1/tasks/:id/tags
type AddTagToTaskRequest struct {
	TagID int `json:"tag_id" binding:"required,min=1"`
}

func (r *AddTagToTaskRequest) Validate() error {
	if r.TagID < 1 {
		return &utils.ValidationError{Field: "tag_id", Message: "tag_id is required"}
	}
	return nil
}
