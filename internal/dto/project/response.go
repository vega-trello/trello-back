package dto

import (
	"time"

	"github.com/vega-trello/trello-back/internal/model"
)

type ProjectResponse struct {
	UUID        string  `json:"uuid"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func FromModel(p *model.Project) *ProjectResponse {
	if p == nil {
		return nil
	}
	return &ProjectResponse{
		UUID:        p.UUID.String(),
		Title:       p.Title,
		Description: p.Description,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
	}
}

func FromModels(projects []*model.Project) []*ProjectResponse {
	responses := make([]*ProjectResponse, 0, len(projects))
	for _, p := range projects {
		responses = append(responses, FromModel(p))
	}
	return responses
}
