package dto

import (
	"time"
)

type TagResponse struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Color     int       `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

// Ответ на GET /api/v1/projects/:id/tags
type TagListResponse struct {
	Tags  []TagResponse `json:"tags"`
	Total int           `json:"total"`
}
