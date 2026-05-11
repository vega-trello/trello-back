package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/vega-trello/trello-back/internal/model"
)

type ColumnResponse struct {
	ID          int       `json:"id"`
	ProjectUUID uuid.UUID `json:"project_uuid"`
	Name        string    `json:"name"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	// UpdatedAt удалён — нет в схеме
}

// FromModel конвертирует model.Column - ColumnResponse
func FromModel(c *model.Column) ColumnResponse {
	return ColumnResponse{
		ID:          c.ID,
		ProjectUUID: c.ProjectUUID,
		Name:        c.Name,
		Position:    c.Position,
		CreatedAt:   c.CreatedAt,
	}
}

func FromModels(columns []*model.Column) []ColumnResponse {
	if columns == nil {
		return []ColumnResponse{}
	}
	res := make([]ColumnResponse, len(columns))
	for i, c := range columns {
		res[i] = FromModel(c)
	}
	return res
}
