package dto

import (
	"time"
)

type MemberResponse struct {
	Username    string    `json:"username"`
	UUID        string    `json:"uuid"`
	ProjectUUID string    `json:"project_uuid"`
	RoleID      int       `json:"role_id"`
	JoinedAt    time.Time `json:"joined_at"`
	RoleName    string    `json:"role_name,omitempty"`
}

type MemberListResponse struct {
	Members []MemberResponse `json:"members"`
	Total   int              `json:"total"`
}
