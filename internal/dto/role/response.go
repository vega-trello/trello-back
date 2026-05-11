// internal/dto/role/response.go
package dto

import (
	"github.com/google/uuid"
	"github.com/vega-trello/trello-back/internal/model"
)

type RoleResponse struct {
	ID          int        `json:"id"`
	ProjectUUID *uuid.UUID `json:"project_uuid"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
}

// FromModel конвертирует model.Role - RoleResponse
func FromModel(r *model.Role) RoleResponse {
	return RoleResponse{
		ID:          r.ID,
		ProjectUUID: r.ProjectUUID,
		Name:        r.Name,
		Description: r.Description,
	}
}

// FromModels конвертирует срез моделей
func FromModels(roles []*model.Role) []RoleResponse {
	if roles == nil {
		return []RoleResponse{}
	}
	res := make([]RoleResponse, len(roles))
	for i, r := range roles {
		res[i] = FromModel(r)
	}
	return res
}

type PermissionResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// FromPermissionModel конвертирует модель разрешения
func FromPermissionModel(p *model.Permission) PermissionResponse {
	return PermissionResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
	}
}
