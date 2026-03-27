package dto

import (
	"time"
)

type ColumnResponse struct {
	ID          int       `json:"id"`
	ProjectUUID string    `json:"projectUuid"`
	Name        string    `json:"name"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	TaskCount   int       `json:"task_count,omitempty"`
}

type ColumnListResponse struct {
	Columns []ColumnResponse `json:"columns"`
	Total   int              `json:"total"`
}
