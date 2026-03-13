package model

import (
	"time"

	"github.com/google/uuid"
)

type Column struct {
	ID          int       `db:"id"`
	ProjectUUID uuid.UUID `db:"project_uuid"`
	Name        string    `db:"name"`
	Position    int       `db:"position"`
	CreatedAt   time.Time `db:"created_at"`
}
