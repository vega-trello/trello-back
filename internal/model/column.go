package model

import (
	"time"

	"github.com/google/uuid"
)

type Column struct {
	ID          int       `db:"id" json:"id"`
	ProjectUUID uuid.UUID `db:"project_uuid" json:"project_uuid"`
	Name        string    `db:"name" json:"name"` // VARCHAR(64)
	Position    int       `db:"position" json:"position"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}
