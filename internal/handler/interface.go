package handler

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	dto_col "github.com/vega-trello/trello-back/internal/dto/column"
	dto_member "github.com/vega-trello/trello-back/internal/dto/member"
	dto_task "github.com/vega-trello/trello-back/internal/dto/task"
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

type TaskServiceInterface interface {
	CreateTask(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto_task.CreateTaskRequest) (*model.TaskDB, error)
	GetProjectTasks(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, archived *bool) ([]*model.TaskDB, error)
	GetTask(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) (*model.TaskDB, error)
	UpdateTask(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, req dto_task.UpdateTaskRequest) (*model.TaskDB, error)
	DeleteTask(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) error
	MoveTask(ctx context.Context, projectUUID uuid.UUID, taskID int, targetColumnID int, userUUID uuid.UUID) error
	ArchiveTask(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, archive bool) error
}

type MemberServiceInterface interface {
	GetProjectMembers(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*dto_member.MemberResponse, error)
	GetMember(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, targetUserUUID uuid.UUID) (*dto_member.MemberResponse, error)
	AddMember(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto_member.CreateMemberRequest) (*model.ProjectMember, error)
	UpdateMemberRole(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, targetUserUUID uuid.UUID, req dto_member.UpdateMemberRequest) (*model.ProjectMember, error)
	RemoveMember(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, targetUserUUID uuid.UUID) error
}

type JWTManagerInterface interface {
	Generate(userUUID uuid.UUID) (string, error)
	Parse(token string) (uuid.UUID, error)
}
