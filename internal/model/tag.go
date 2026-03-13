package model

import (
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	ID          int       `db:"id"`
	ProjectUUID uuid.UUID `db:"project_uuid"`
	Name        string    `db:"name"`
	Color       int       `db:"color"`
	CreatedAt   time.Time `db:"created_at"`
}

type TaskTag struct {
	TaskID  int       `db:"task_id"`
	TagID   int       `db:"tag_id"`
	AddedAt time.Time `db:"added_at"`
}
