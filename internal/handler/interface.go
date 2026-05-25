package handler

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	dto_assignee "github.com/vega-trello/trello-back/internal/dto/assignee"
	dto_col "github.com/vega-trello/trello-back/internal/dto/column"
	dto_member "github.com/vega-trello/trello-back/internal/dto/member"
	dto_status "github.com/vega-trello/trello-back/internal/dto/status"
	dto_tag "github.com/vega-trello/trello-back/internal/dto/tag"
	dto_task "github.com/vega-trello/trello-back/internal/dto/task"
	"github.com/vega-trello/trello-back/internal/model"
)

type UserServiceInterface interface {
	Register(ctx context.Context, username, password string) (*model.User, error)
	Login(ctx context.Context, username, password string) (*model.User, error)
	LoginBySSO(ctx context.Context, provider, extID, username string, metadata json.RawMessage) (*model.User, error)
	GetSelfProfile(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error)
	UpdateSelfProfile(ctx context.Context, userUUID uuid.UUID, oldPass, newName, newPass string) (*model.SelfUser, error)
	GetOtherUserProfile(ctx context.Context, targetUserUUID uuid.UUID) (*model.User, error)
	Logout(ctx context.Context, userUUID uuid.UUID) error
}

type ProjectServiceInterface interface {
	GetUserProjects(ctx context.Context, userUUID uuid.UUID) ([]*model.Project, error)
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

type AssigneeServiceInterface interface {
	GetTaskAssignees(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) ([]*dto_assignee.AssigneeResponse, error)
	AssignUserToTask(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, req dto_assignee.CreateAssigneeRequest) error
	RemoveAssignee(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, assigneeUUID uuid.UUID) error
}

type TagServiceInterface interface {
	GetProjectTags(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Tag, error)
	GetTaskTags(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) ([]*model.Tag, error)
	CreateTag(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto_tag.CreateTagRequest) (*model.Tag, error)
	UpdateTag(ctx context.Context, tagID int, userUUID uuid.UUID, req dto_tag.UpdateTagRequest) (*model.Tag, error)
	DeleteTag(ctx context.Context, tagID int, userUUID uuid.UUID) error
	AddTagToTask(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, taskID int, tagID int) error
	RemoveTagFromTask(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, taskID int, tagID int) error
}

type RoleServiceInterface interface {
	CreateRole(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error)
	GetProjectRoles(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Role, error)
	GetRole(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) (*model.Role, error)
	UpdateRole(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error)
	DeleteRole(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) error
	GetRolePermissions(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) ([]*model.Permission, error)
}

type StatusServiceInterface interface {
	CreateStatus(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto_status.CreateStatusRequest) (*model.ProjectStatus, error)
	GetProjectStatuses(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.ProjectStatus, error)
	GetStatus(ctx context.Context, projectUUID uuid.UUID, statusID int, userUUID uuid.UUID) (*model.ProjectStatus, error)
	UpdateStatus(ctx context.Context, projectUUID uuid.UUID, statusID int, userUUID uuid.UUID, req dto_status.UpdateStatusRequest) (*model.ProjectStatus, error)
	DeleteStatus(ctx context.Context, projectUUID uuid.UUID, statusID int, userUUID uuid.UUID) error
}

type JWTManagerInterface interface {
	Generate(userUUID uuid.UUID) (string, error)
	Parse(token string) (uuid.UUID, error)
}
