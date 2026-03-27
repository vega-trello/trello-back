package dto

import (
	"time"
)

type TaskResponse struct {
	ID          int        `json:"id"`
	ColumnID    int        `json:"column_id"`
	CreatorUUID string     `json:"creator_uuid"`
	Tags        []int      `json:"tags"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// GET /projects/{projectUUID}/tasks
type TaskListResponse struct {
	Tasks []TaskResponse `json:"tasks"`
	Total int            `json:"total"`
}

// / get: projects/{projectUUID}/task?taskID={id}
type TaskDetailResponse struct {
	TaskResponse
	ColumnName      string         `json:"column_name,omitempty"`
	CreatorUsername string         `json:"creator_username,omitempty"`
	Assignees       []AssigneeInfo `json:"assignees,omitempty"`
	TagsDetail      []TagInfo      `json:"tags_detail,omitempty"`
}

type AssigneeInfo struct {
	UserUUID string `json:"user_uuid"`
	Username string `json:"username"`
}

type TagInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color int    `json:"color"`
}
