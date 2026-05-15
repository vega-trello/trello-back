package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/member"
	dtoProject "github.com/vega-trello/trello-back/internal/dto/project"
	"github.com/vega-trello/trello-back/internal/model"
)

type UserRepository interface {
	// Auth endpoints
	RegisterPasswordUser(ctx context.Context, username string, passwordHash []byte) (*model.User, error)
	FindUserByUsername(ctx context.Context, username string) (*model.User, []byte, error)
	FindOrCreateUserBySSO(ctx context.Context, provider, externalID, username string) (*model.User, error)
	FindUserByUUID(ctx context.Context, userUUID uuid.UUID) (*model.User, error)
	Logout(ctx context.Context, userUUID uuid.UUID) error

	// User endpoints
	GetSelfUser(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error)
	UpdateSelfUser(ctx context.Context, userUUID uuid.UUID, oldPassword, newUsername, newPassword string) (*model.SelfUser, error)
}

type RoleRepository interface {
	Create(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error)
	FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Role, error)
	FindByID(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) (*model.Role, error)
	Update(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID, name string, description *string, permissionIDs []int) (*model.Role, error)
	Delete(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) error
	FindPermissions(ctx context.Context, projectUUID uuid.UUID, roleID int, userUUID uuid.UUID) ([]*model.Permission, error)
}

type ProjectRepository interface {
	Create(ctx context.Context, userUUID uuid.UUID, req dtoProject.CreateProjectRequest) (*model.Project, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
	FindByUser(ctx context.Context, userUUID uuid.UUID) ([]*model.Project, error)
	Update(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, title *string, description *string) (*model.Project, error)
	Delete(ctx context.Context, projectUUID uuid.UUID) error
	IsMember(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) (bool, error)
	IsOwner(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) (bool, error)
}

type StatusRepository interface {
	Create(ctx context.Context, projectUUID uuid.UUID, name string, callerUUID uuid.UUID) (*model.ProjectStatus, error)
	FindByProject(ctx context.Context, projectUUID uuid.UUID, callerUUID uuid.UUID) ([]*model.ProjectStatus, error)
	FindByID(ctx context.Context, projectUUID uuid.UUID, statusID int, callerUUID uuid.UUID) (*model.ProjectStatus, error)
	Update(ctx context.Context, projectUUID uuid.UUID, statusID int, newName string, callerUUID uuid.UUID) (*model.ProjectStatus, error)
	Delete(ctx context.Context, projectUUID uuid.UUID, statusID int, callerUUID uuid.UUID) error
}

type ColumnRepository interface {
	Create(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, name string, position *int) (*model.Column, error)
	FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Column, error)
	FindByID(ctx context.Context, columnID int, userUUID uuid.UUID) (*model.Column, error)
	Update(ctx context.Context, columnID int, userUUID uuid.UUID, name string, position *int) (*model.Column, error)
	Delete(ctx context.Context, columnID int, userUUID uuid.UUID) error
	Move(ctx context.Context, columnID int, userUUID uuid.UUID, direction string) (*model.Column, error)
}

type TaskRepository interface {
	Create(ctx context.Context, projectUUID uuid.UUID, columnID int, statusID *int, creatorUUID uuid.UUID, title string, description string, startDate *time.Time, endDate *time.Time) (*model.TaskDB, error)
	FindByID(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) (*model.TaskDB, error)
	FindByProjectUUID(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, archived *bool) ([]*model.TaskDB, error)
	FindByColumn(ctx context.Context, columnID int, userUUID uuid.UUID) ([]*model.TaskDB, error)
	Update(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, title *string, description *string, startDate *time.Time, endDate *time.Time, columnID *int, statusID *int, archived *bool) (*model.TaskDB, error)
	Delete(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID) error
	Move(ctx context.Context, projectUUID uuid.UUID, taskID int, targetColumnID int, userUUID uuid.UUID) error
	Archive(ctx context.Context, projectUUID uuid.UUID, taskID int, userUUID uuid.UUID, archive bool) error
}

type MemberRepository interface {
	Create(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, roleID int) (*model.ProjectMember, error)
	FindByProjectUUIDWithDetails(ctx context.Context, projectUUID uuid.UUID) ([]*dto.MemberResponse, error)
	FindByProjectAndUser(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) (*model.ProjectMember, error)
	Update(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, roleID int) (*model.ProjectMember, error)
	Delete(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) error
}
