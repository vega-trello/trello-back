package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/vega-trello/trello-back/internal/model"
)

type TaskResponse struct {
	ID          int        `json:"id"`
	ColumnID    int        `json:"column_id"`
	StatusID    *int       `json:"status_id"`
	CreatorUUID uuid.UUID  `json:"creator_uuid"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	StartDate   *time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ArchivedAt  *time.Time `json:"archived_at"`
}

func FromModel(t *model.TaskDB) TaskResponse {
	return TaskResponse{
		ID:          t.ID,
		ColumnID:    t.ColumnID,
		StatusID:    t.StatusID,
		CreatorUUID: t.CreatorUUID,
		Title:       t.Title,
		Description: t.Description,
		StartDate:   t.StartDate,
		EndDate:     t.EndDate,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		ArchivedAt:  t.ArchivedAt,
	}
}

func FromModels(tasks []*model.TaskDB) []TaskResponse {
	if tasks == nil {
		return []TaskResponse{}
	}
	res := make([]TaskResponse, len(tasks))
	for i, t := range tasks {
		res[i] = FromModel(t)
	}
	return res
}

type TaskDetailResponse struct {
	TaskResponse
	ColumnName      string         `json:"column_name,omitempty"`
	CreatorUsername string         `json:"creator_username,omitempty"`
	Assignees       []AssigneeInfo `json:"assignees,omitempty"`
	TagsDetail      []TagInfo      `json:"tags_detail,omitempty"`
}

type AssigneeInfo struct {
	UserUUID uuid.UUID `json:"user_uuid"`
	Username string    `json:"username"`
}

type TagInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}
