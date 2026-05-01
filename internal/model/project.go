package model

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	UUID        uuid.UUID `db:"uuid"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
