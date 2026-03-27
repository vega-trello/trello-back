package dto

import (
	"github.com/vega-trello/trello-back/internal/utils"
)

// post: /projects
type CreateProjectRequest struct {
	Title       string `json:"title" binding:"required,min=1,max=128"`
	Description string `json:"description,omitempty" binding:"omitempty,max=512"`
}

// patch: /projects/{projectUID}
type UpdateProjectRequest struct {
	Title       string `json:"title" binding:"omitempty,min=1,max=128"`
	Description string `json:"description,omitempty" binding:"omitempty,max=512"`
}

func (r *CreateProjectRequest) Validate() error {
	if r.Title == "" {
		return &utils.ValidationError{Field: "title", Message: "title is required"}
	}
	return nil
}

func (r *UpdateProjectRequest) Validate() error {
	if r.Title != "" && len(r.Title) > 128 {
		return &utils.ValidationError{Field: "title", Message: "title must be at most 128 characters"}
	}
	return nil
}
