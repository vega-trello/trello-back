package dto

import (
	"time"
)

type RoleResponse struct {
	ID          int       `json:"id"`
	ProjectUUID string    `json:"projectUUID"`
	Discription string    `json:"discription"`
	CreatedAt   time.Time `json:"createdAt"`
}

type PermissionResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RoleListResponse struct {
	Roles []RoleResponse `json:"roles"`
	Total int            `json:"total"`
}

type PermissionListRespoonse struct {
	Permissions []PermissionResponse `json:"permissions"`
	Total       int                  `json:"total"`
}

type RoleDetailResponse struct {
	RoleResponse
	Permissions []PermissionResponse `json:"permissions,omitempty"`
}
