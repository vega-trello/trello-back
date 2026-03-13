package dto

import (
	"time"
)

type StatusResponse struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Ответ на GET /api/v1/projects/:id/statuses
type StatusListResponse struct {
	Statuses []StatusResponse `json:"statuses"`
	Total    int              `json:"total"`
}
