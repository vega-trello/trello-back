package dto

import (
	"errors"
	"regexp"
)

// hexColorRegex  ^#[0-9A-Fa-f]{6}$ регулярка
var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type CreateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func (r CreateTagRequest) Validate() error {
	if r.Name == "" || len(r.Name) > 32 {
		return errors.New("name must be between 1 and 32 characters")
	}
	if !hexColorRegex.MatchString(r.Color) {
		return errors.New("color must match pattern #RRGGBB (e.g., #FF0000)")
	}
	return nil
}

// UpdateTagRequest соответствует схеме UpdateTag в OpenAPI
// По контракту оба поля обязательны при обновлении
type UpdateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Validate переиспользует логику CreateTagRequest
func (r UpdateTagRequest) Validate() error {
	return CreateTagRequest(r).Validate()
}
