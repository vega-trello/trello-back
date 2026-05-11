package dto

import "github.com/vega-trello/trello-back/internal/utils"

// POST /projects/{projectUUID}/task
type CreateTaskRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
	ColumnID    *int    `json:"column_id"`
}

// PUT /api/v1/tasks/:id
type UpdateTaskRequest struct {
	Title       *string `json:"title" binding:"omitempty,min=1,max=256"`
	Description *string `json:"description" binding:"omitempty,max=2048"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
	ColumnID    *int    `json:"column_id,omitempty"`
	Archived    *bool   `json:"archived"`
}

// DELETE /projects/{projectUUID}/task?taskID={id}
type DeleteTaskRequest struct {
	ID int `json:"id" binding:"required,min=1"`
}

// POST /projects/{projectUUID}/task/tags
type AddTagToTaskRequest struct {
	TaskID int `json:"task_id" binding:"required,min=1"`
	TagID  int `json:"tag_id" binding:"required,min=1"`
}

// DELETE /projects/{projectUUID}/task/tags
type DeleteTagFromTaskRequest struct {
	TaskID int `json:"task_id" binding:"required,min=1"`
	TagID  int `json:"tag_id" binding:"required,min=1"`
}

func (r *CreateTaskRequest) Validate() error {
	if r.Title == "" || len(r.Title) > 256 {
		return &utils.ValidationError{
			Field:   "title",
			Message: "title must be between 1 and 256 characters",
		}
	}
	if r.Description != "" && len(r.Description) > 2048 {
		return &utils.ValidationError{
			Field:   "description",
			Message: "description must not exceed 2048 characters",
		}
	}
	return nil
}

func (r *UpdateTaskRequest) Validate() error {
	if r.Title != nil && (len(*r.Title) == 0 || len(*r.Title) > 256) {
		return &utils.ValidationError{
			Field:   "title",
			Message: "title must be between 1 and 256 characters",
		}
	}
	if r.Description != nil && len(*r.Description) > 2048 {
		return &utils.ValidationError{
			Field:   "description",
			Message: "description must not exceed 2048 characters",
		}
	}
	return nil
}

func (r *DeleteTaskRequest) Validate() error {
	if r.ID < 1 {
		return &utils.ValidationError{
			Field:   "id",
			Message: "id is required and must be positive",
		}
	}
	return nil
}

func (r *DeleteTagFromTaskRequest) Validate() error {
	if r.TaskID < 1 {
		return &utils.ValidationError{
			Field:   "task_id",
			Message: "task_id is required",
		}
	}
	if r.TagID < 1 {
		return &utils.ValidationError{
			Field:   "tag_id",
			Message: "tag_id is required",
		}
	}
	return nil
}
