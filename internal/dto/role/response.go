// internal/dto/role/response.go
package dto

import (
	"time"

	"github.com/google/uuid"
)

// RoleResponse — ответ с информацией о роли
type RoleResponse struct {
	ID          int       `json:"id"`
	ProjectUUID uuid.UUID `json:"project_uuid"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// PermissionResponse — ответ с информацией о разрешении
type PermissionResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RoleWithPermissionsResponse — роль с списком разрешений
type RoleWithPermissionsResponse struct {
	RoleResponse
	Permissions []PermissionResponse `json:"permissions"`
}
