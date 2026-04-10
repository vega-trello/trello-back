package model

import "github.com/google/uuid"

type Role struct {
	ID          int       `db:"id"`
	ProjectUUID uuid.UUID `db:"project_uuid"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
}
type Permission struct {
	ID          int    `db:"id"`
	Name        string `db:"name"`
	Description string `db:"description"`
}

type RolePermission struct {
	RoleID       int `db:"role_id"`
	PermissionID int `db:"permission_id"`
}

// 1=owner, 2=admin, 3=member, 4=viewer
const (
	RoleOwner  = 1
	RoleAdmin  = 2
	RoleMember = 3
	RoleViewer = 4
)
