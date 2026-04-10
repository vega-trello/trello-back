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
