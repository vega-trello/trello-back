package dto

import (
	"errors"
	"regexp"

	"github.com/vega-trello/trello-back/internal/utils"
)

// POST /api/v1/projects/:id/columns
type CreateColumnRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=64"`
	Position *int   `json:"position" binding:"omitempty,min=0"`
}

// PATCH /api/v1/columns/:id
type UpdateColumnRequest struct {
	Name     string  `json:"name" binding:"required,min=1,max=64"` // 🔹 string, required
	Color    *string `json:"color" binding:"omitempty"`            // 🔹 опционален
	Position *int    `json:"position,omitempty" binding:"omitempty,min=0"`
}

type MoveColumnRequest struct {
	Direction string `json:"direction" binding:"required,oneof=left right"`
}

func (r *MoveColumnRequest) Validate() error {
	if r.Direction != "left" && r.Direction != "right" {
		return errors.New("direction must be 'left' or 'right'")
	}
	return nil
}

func (r *CreateColumnRequest) Validate() error {
	if r.Name == "" || len(r.Name) > 64 {
		return &utils.ValidationError{
			Field:   "name",
			Message: "name must be between 1 and 64 characters",
		}
	}
	return nil
}

func (r *UpdateColumnRequest) Validate() error {
	if r.Name == "" || len(r.Name) > 64 {
		return &utils.ValidationError{
			Field:   "name",
			Message: "name is required and must be between 1 and 64 characters",
		}
	}
	if r.Color != nil && !isValidHexColor(*r.Color) {
		return &utils.ValidationError{Field: "color", Message: "color must be a valid HEX string (#RRGGBB)"}
	}
	return nil
}

func isValidHexColor(s string) bool {
	matched, _ := regexp.MatchString(`^#([0-9A-Fa-f]{6}|[0-9A-Fa-f]{8})$`, s)
	return matched
}
