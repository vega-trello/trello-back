package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/vega-trello/trello-back/internal/model"
)

type TagResponse struct {
	ID          int       `json:"id"`
	ProjectUUID uuid.UUID `json:"project_uuid"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func FromModel(t *model.Tag) TagResponse {
	return TagResponse{
		ID:          t.ID,
		ProjectUUID: t.ProjectUUID,
		Name:        t.Name,
		Color:       t.Color,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func FromModels(tags []*model.Tag) []TagResponse {
	if tags == nil {
		return []TagResponse{}
	}
	res := make([]TagResponse, len(tags))
	for i, t := range tags {
		res[i] = FromModel(t)
	}
	return res
}
