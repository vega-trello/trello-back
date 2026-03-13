package model

import (
	"time"

	"github.com/google/uuid"
)

type ProjectStatus struct {
	ID        int       `db:"id"`
	ProjectID uuid.UUID `db:"project_id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}
