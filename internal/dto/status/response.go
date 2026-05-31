package dto

import (
	"github.com/vega-trello/trello-back/internal/model"
)

type StatusResponse struct {
	ID          int    `json:"id"`
	ProjectUUID string `json:"project_uuid"`
	Name        string `json:"name"`
	CreatedAt   string `json:"created_at"`
}

func FromModel(s *model.ProjectStatus) StatusResponse {
	return StatusResponse{
		ID:          s.ID,
		ProjectUUID: s.ProjectUUID.String(),
		Name:        s.Name,
		CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func FromModels(statuses []*model.ProjectStatus) []StatusResponse {
	if statuses == nil {
		return []StatusResponse{}
	}
	res := make([]StatusResponse, len(statuses))
	for i, s := range statuses {
		res[i] = FromModel(s)
	}
	return res
}
