package model

import (
	"time"

	"github.com/google/uuid"
)

type TaskDB struct {
	ID          int        `db:"id" json:"id"`
	ColumnID    int        `db:"column_id" json:"column_id"`
	StatusID    *int       `db:"status_id" json:"status_id"`
	CreatorUUID uuid.UUID  `db:"creator_uuid" json:"creator_uuid"`
	Title       string     `db:"title" json:"title"`
	Description string     `db:"description" json:"description"`
	Color       *string    `db:"color" json:"color"`
	StartDate   *time.Time `db:"start_date" json:"start_date"`
	EndDate     *time.Time `db:"end_date" json:"end_date"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
	Done        bool       `db:"done" json:"done"`
	ArchivedAt  *time.Time `db:"archived_at" json:"archived_at"`
}

type TaskAssignee struct {
	TaskID     int       `db:"task_id" json:"task_id"`
	UserUUID   uuid.UUID `db:"user_uuid" json:"user_uuid"`
	AssignedAt time.Time `db:"assigned_at" json:"assigned_at"`
}
