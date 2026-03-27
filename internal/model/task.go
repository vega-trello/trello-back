package model

import (
	"time"

	"github.com/google/uuid"
)

type TaskDB struct {
	ID          int        `db:"id"`
	ColumnID    int        `db:"column_id"`
	StatusID    *int       `db:"status_id"` // nullable
	CreatorUUID uuid.UUID  `db:"creator_uuid"`
	Title       string     `db:"title"`
	Description string     `db:"description"`
	DeletedAt   *time.Time `db:"delete_at"`   // nullable
	ArchivedAt  *time.Time `db:"archived_at"` // nullable
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	StartDate   *time.Time `db:"start_date"` // nullable
	EndDate     *time.Time `db:"end_date"`   // nullable
}

type TaskAssignee struct {
	TaskID     int       `db:"task_id"`
	UserUUID   uuid.UUID `db:"user_uuid"`
	AssignedAt time.Time `db:"assigned_at"`
}
