package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	UUID     uuid.UUID `db:"uuid" json:"uuid"`
	Username string    `db:"username" json:"username"`
}

// Используется для GET /user и PATCH /user
type SelfUser struct {
	User
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	UserType  string    `db:"user_type" json:"user_type"` // "manual" или "sso"
}

// ManualUser соответствует таблице manual_user (внутренняя)
type ManualUser struct {
	UserUUID     uuid.UUID `db:"user_uuid"`
	PasswordHash []byte    `db:"password_hash"`
}

// SsoUser соответствует таблице sso_user (внутренняя)
type SsoUser struct {
	UserUUID   uuid.UUID       `db:"user_uuid"`
	Provider   string          `db:"provider"`
	ExternalID string          `db:"external_id"`
	Metadata   json.RawMessage `db:"metadata"`
}
