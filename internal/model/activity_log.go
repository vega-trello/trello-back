package model

import (
	"time"

	"github.com/google/uuid"
)

type ActivityLogDB struct {
	ID          int        `db:"id"`
	ProjectUUID uuid.UUID  `db:"project_uuid"`
	UserUUID    uuid.UUID  `db:"user_uuid"`
	ActionType  string     `db:"action_type"`
	EntityType  *string    `db:"entity_type"` // nullable
	EntityUUID  *uuid.UUID `db:"entity_uuid"` // nullable
	Metadata    []byte     `db:"metadata"`    // nullable
	CreatedAt   time.Time  `db:"created_at"`
}
