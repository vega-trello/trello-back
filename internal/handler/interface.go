// internal/handler/interfaces.go
package handler

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
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

type JWTManagerInterface interface {
	Generate(userUUID uuid.UUID) (string, error)
	Parse(token string) (uuid.UUID, error)
}
