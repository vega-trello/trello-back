package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	UUID      uuid.UUID `db:"uuid"`
	Username  string    `db:"username"`
	UserType  string    `db:"user_type"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type ManualUser struct {
	UserUUID     uuid.UUID `db:"user_uuid"`
	PasswordHash []byte    `db:"password_hash"`
}

type SsoUser struct {
	UserUUID   uuid.UUID       `db:"user_uuid"`
	Provider   string          `db:"provider"`
	ExternalID string          `db:"external_id"`
	Metadata   json.RawMessage `db:"metadata"`
}
