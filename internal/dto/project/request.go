package dto

import (
	"github.com/vega-trello/trello-back/internal/utils"
)

// запрос на создание проекта (POST /projects)
type CreateProjectRequest struct {
	Title       string  `json:"title" binding:"required,min=1,max=128"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=512"`
}

func (r *CreateProjectRequest) Validate() error {
	if r.Description != nil && len(*r.Description) > 512 {
		return &utils.ValidationError{Field: "description", Message: "description must be at most 512 characters"}
	}
	return nil
}

// запрос на обновление проекта (PATCH /projects/{uuid})
// Все поля опциональны (*string), чтобы реализовать частичное обновление.
type UpdateProjectRequest struct {
	Title       *string `json:"title,omitempty" binding:"omitempty,min=1,max=128"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=512"`
}

func (r *UpdateProjectRequest) Validate() error {
	if r.Title != nil {
		if *r.Title == "" || len(*r.Title) > 128 {
			return &utils.ValidationError{Field: "title", Message: "title must be between 1 and 128 characters"}
		}
	}

	if r.Description != nil && len(*r.Description) > 512 {
		return &utils.ValidationError{Field: "description", Message: "description must be at most 512 characters"}
	}
	return nil
}
