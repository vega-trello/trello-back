package dto

import (
	"time"
)

// post /projects 200
type ProjectResponse struct {
	UUID        string    `json:"uuid"`
	Title       string    `json:"title" binding:"required,min=1,max=128"`
	Description string    `json:"description,omitempty" binding:"omitempty,max=512"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type ProjectListResponse struct {
	Projects []ProjectResponse `json:"projects"`
	Total    int               `json:"total"`
}
