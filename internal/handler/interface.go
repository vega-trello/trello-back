package handler

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	dto_col "github.com/vega-trello/trello-back/internal/dto/column"
	"github.com/vega-trello/trello-back/internal/model"
)

type UserServiceInterface interface {
	Register(ctx context.Context, username, password string) (*model.User, error)
	Login(ctx context.Context, username, password string) (*model.User, error)
	LoginBySSO(ctx context.Context, provider, extID, username string, metadata json.RawMessage) (*model.User, error)
	GetProfile(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error)
	UpdateProfile(ctx context.Context, userUUID uuid.UUID, oldPass, newName, newPass string) (*model.SelfUser, error)
	Logout(ctx context.Context, userUUID uuid.UUID) error
}

type ProjectServiceInterface interface {
	GetUserProjects(ctx context.Context, userUUID uuid.UUID) ([]*model.Project, error) // 🔹 GetUserProjects, не ListUserProjects
	CreateProject(ctx context.Context, creatorUUID uuid.UUID, title string, description *string) (*model.Project, error)
	GetProject(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) (*model.Project, error)
	UpdateProject(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, title *string, description *string) (*model.Project, error)
	DeleteProject(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) error
}

type ColumnServiceInterface interface {
	CreateColumn(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto_col.CreateColumnRequest) (*model.Column, error)
	GetProjectColumns(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Column, error)
	GetColumn(ctx context.Context, columnID int, userUUID uuid.UUID) (*model.Column, error)
	UpdateColumn(ctx context.Context, columnID int, userUUID uuid.UUID, req dto_col.UpdateColumnRequest) (*model.Column, error)
	DeleteColumn(ctx context.Context, columnID int, userUUID uuid.UUID) error
	MoveColumn(ctx context.Context, columnID int, userUUID uuid.UUID, direction string) (*model.Column, error)
}

type JWTManagerInterface interface {
	Generate(userUUID uuid.UUID) (string, error)
	Parse(token string) (uuid.UUID, error)
}
