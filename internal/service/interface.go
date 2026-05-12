package service

import (
	"context"

	"github.com/google/uuid"
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
