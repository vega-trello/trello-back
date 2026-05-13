package model

import (
	"time"

	"github.com/google/uuid"
)

// - nil: поле не передано (не обновлять при PATCH)
// - &"": передана пустая строка (явно очистить поле)
type Project struct {
	UUID        uuid.UUID `db:"uuid" json:"uuid"`
	Title       string    `db:"title" json:"title"`
	Description *string   `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}
