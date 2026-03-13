package dto

import "time"

type AuthResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

type UserInfo struct {
	UUID      string    `json:"uuid"`
	Username  string    `json:"username"`
	UserType  string    `json:"user_type"`
	CreatedAt time.Time `json:"created_at"`
}
