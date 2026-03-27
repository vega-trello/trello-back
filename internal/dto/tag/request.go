package dto

import (
	"github.com/vega-trello/trello-back/internal/utils"
)

// POST /api/v1/projects/:id/tags
type CreateTagRequest struct {
	Name  string `json:"name" binding:"required,min=1,max=32"`
	Color int    `json:"color" binding:"required"`
}

// PUT /api/v1/tags/:id
type UpdateTagRequest struct {
	Name  string `json:"name" binding:"omitempty,min=1,max=32"`
	Color *int   `json:"color,omitempty" binding:"omitempty,min=0,max=16777215"`
}

type AttachTagRequest struct {
	TagID int `json:"tag_id" binding:"required,min=1"`
}

type DeleteTagFromTaskRequest struct {
	TaskID int `json:"tag_id" binding:"required,min=1"`
	TagID  int `json:"tag_id" binding:"required,min=1"`
}

func (r *CreateTagRequest) Validate() error {
	if r.Name == "" {
		return &utils.ValidationError{Field: "name", Message: "name is required"}
	}
	return nil
}

func (r *UpdateTagRequest) Validate() error {
	if r.Name != "" && len(r.Name) > 32 {
		return &utils.ValidationError{
			Field:   "name",
			Message: "name must be at most 32 characters",
		}
	}

	if r.Color != nil && (*r.Color < 0 || *r.Color > 16777215) {
		return &utils.ValidationError{
			Field:   "color",
			Message: "color must be between 0 and 16777215 (0x000000-0xFFFFFF)",
		}
	}

	return nil
}

func (r *AttachTagRequest) Validate() error {
	if r.TagID < 1 {
		return &utils.ValidationError{
			Field:   "tag_id",
			Message: "tag_id is required and must be positive",
		}
	}
	return nil
}

func (r *DeleteTagFromTaskRequest) Validate() error {
	if r.TaskID < 1 {
		return &utils.ValidationError{
			Field:   "task_id",
			Message: "task_id is required and must be positive",
		}
	}
	if r.TagID < 1 {
		return &utils.ValidationError{
			Field:   "tag_id",
			Message: "tag_id is required and must be positive",
		}
	}
	return nil
}
