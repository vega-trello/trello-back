package model

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	UUID        uuid.UUID `db:"uuid"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	CreateAt    time.Time `db:"create_at"`
	UpdateAt    time.Time `db:"update_at"`
}
