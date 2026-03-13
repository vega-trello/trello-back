package dto

import (
	"time"
)

type MemberResponse struct {
	UserUUID string    `json:"user_uuid"`
	Username string    `json:"username"`
	RoleID   int       `json:"role_id"`
	RoleName string    `json:"role_name"`
	JoinedAt time.Time `json:"joined_at"`
}

// Ответ на GET /api/v1/projects/:id/members
type MemberListResponse struct {
	Members []MemberResponse `json:"members"`
	Total   int              `json:"total"`
}
