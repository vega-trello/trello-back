package dto

import (
	"time"
)

type TagResponse struct {
	ID          int       `json:"id"`
	ProjectUUID string    `json:"project_uuid"`
	Name        string    `json:"name"`
	Color       int       `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Ответ на GET /api/v1/projects/:id/tags
type TagListResponse struct {
	Tags  []TagResponse `json:"tags"`
	Total int           `json:"total"`
}

type TaskTagResponse struct {
	TaskID    int          `json:"task_id"`
	TagID     string       `json:"tag_id"`
	CreatedAt time.Time    `json:"created_at"`
	Tag       *TagResponse `json:"tag"`
}

type TaskListListResponse struct {
	Tags  []TaskTagResponse `json:"tags"`
	Total int               `json:"total"`
}
