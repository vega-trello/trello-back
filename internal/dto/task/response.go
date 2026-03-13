package dto

import (
	"time"
)

type TaskResponse struct {
	ID              int        `json:"id"`
	ColumnID        int        `json:"column_id"`
	ColumnName      string     `json:"column_name,omitempty"`
	StatusID        *int       `json:"status_id,omitempty"`
	StatusName      *string    `json:"status_name,omitempty"`
	CreatorUUID     string     `json:"creator_uuid"`
	CreatorUsername string     `json:"creator_username,omitempty"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Position        int        `json:"position"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ArchivedAt      *time.Time `json:"archived_at,omitempty"`
}

// Ответ на GET /api/v1/columns/:id/tasks
type TaskListResponse struct {
	Tasks []TaskResponse `json:"tasks"`
	Total int            `json:"total"`
}

type AssigneeResponse struct {
	UserUUID   string    `json:"user_uuid"`
	Username   string    `json:"username"`
	AssignedAt time.Time `json:"assigned_at"`
}

// Ответ на GET /api/v1/tasks/:id/assignees
type AssigneeListResponse struct {
	Assignees []AssigneeResponse `json:"assignees"`
	Total     int                `json:"total"`
}
