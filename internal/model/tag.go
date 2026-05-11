package model

import (
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	ID          int       `db:"id" json:"id"`
	ProjectUUID uuid.UUID `db:"project_uuid" json:"project_uuid"`
	Name        string    `db:"name" json:"name"`   // VARCHAR(32) в БД
	Color       string    `db:"color" json:"color"` // VARCHAR(7) HEX: "#FF0000"
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type TaskTag struct {
	TaskID  int       `db:"task_id"`
	TagID   int       `db:"tag_id"`
	AddedAt time.Time `db:"added_at"`
}
