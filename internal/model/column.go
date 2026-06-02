package model

import (
	"time"

	"github.com/google/uuid"
)

type Column struct {
	ID          int       `db:"id" json:"id"`
	ProjectUUID uuid.UUID `db:"project_uuid" json:"project_uuid"`
	Name        string    `db:"name" json:"name"`
	Position    int       `db:"position" json:"position"`
	Color       *string   `db:"color" json:"color"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}
