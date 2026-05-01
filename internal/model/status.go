package model

import (
	"github.com/google/uuid"
	"time"
)

type ProjectStatus struct {
	ID          int       `db:"id"`
	ProjectUUID uuid.UUID `db:"project_uuid"`
	Name        string    `db:"name"`
	CreatedAt   time.Time `db:"created_at"`
}
