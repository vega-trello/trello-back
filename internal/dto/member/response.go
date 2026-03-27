package dto

import (
	"time"
)

type MemberResponse struct {
	Username    string    `json:"username"`
	UUID        string    `json:"user_uuid"`
	ProjectUUID string    `json:"project_uuid"`
	RoleID      int       `json:"role_id"`
	RoleName    string    `json:"role_name"`
	JoinedAt    time.Time `json:"joined_at"`
}

// Ответ на GET /api/v1/projects/:id/members
type MemberListResponse struct {
	Members []MemberResponse `json:"members"`
	Total   int              `json:"total"`
}
