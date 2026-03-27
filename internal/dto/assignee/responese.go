package dto

import (
	"time"
)

type AssigneeResponse struct {
	TaskID     int       `json:"task_id"`
	UserUUID   string    `json:"user_uuid"`
	AssignedAt time.Time `json:"assigned_at"`
	User       *UserInfo `json:"user,omitempty"`
}

type UserInfo struct {
	Username string `json:"username"`
	UUID     string `json:"uuid"`
}

type AssingeeResponse struct {
	Assignees []AssigneeResponse `json:"assignees"`
	Total     int                `json:"total"`
}
