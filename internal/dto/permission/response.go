package dto

import "github.com/vega-trello/trello-back/internal/model"

// PermissionResponse представляет право доступа в ответах API
type PermissionResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// FromPermissionModel конвертирует model.Permission в PermissionResponse
func FromPermissionModel(p *model.Permission) PermissionResponse {
	return PermissionResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
	}
}

// FromPermissionModels конвертирует слайс []*model.Permission в слайс PermissionResponse
func FromPermissionModels(permissions []*model.Permission) []PermissionResponse {
	result := make([]PermissionResponse, 0, len(permissions))
	for _, p := range permissions {
		if p != nil {
			result = append(result, FromPermissionModel(p))
		}
	}
	return result
}
