package model

import (
	"time"

	"github.com/google/uuid"
)

type ProjectMember struct {
	ProjectUUID uuid.UUID `db:"project_uuid"`
	UserUUID    uuid.UUID `db:"user_uuid"`
	RoleID      int       `db:"role_id"`
	JoinedAt    time.Time `db:"joined_at"`
}
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
