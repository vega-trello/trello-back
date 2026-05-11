package model

import (
	"github.com/google/uuid"
)

type Role struct {
	ID          int        `db:"id" json:"id"`
	ProjectUUID *uuid.UUID `db:"project_uuid" json:"project_uuid"` // nullable для системных ролей
	Name        string     `db:"name" json:"name"`                 // VARCHAR(32)
	Description *string    `db:"description" json:"description"`   // nullable
}

type RolePermission struct {
	RoleID       int `db:"role_id"`
	PermissionID int `db:"permission_id"`
}

type Permission struct {
	ID          int    `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`
	Description string `db:"description" json:"description"`
}

// 1=owner, 2=admin, 3=member, 4=viewer
const (
	RoleOwner  = 1
	RoleAdmin  = 2
	RoleMember = 3
	RoleViewer = 4
)
