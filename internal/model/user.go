package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	UUID      uuid.UUID
	Username  string
	UserType  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ManualUser struct {
	UUID         uuid.UUID
	HashPassword []byte
}

type SsoUser struct {
	UUID       uuid.UUID
	Provider   string
	ExternalID string
	Metadata   []byte
}
