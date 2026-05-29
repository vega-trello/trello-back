package service

import (
	"context"
	"fmt"

	"github.com/vega-trello/trello-back/internal/model"
)

type PermissionService struct {
	repo PermissionRepository
}

func NewPermissionService(repo PermissionRepository) *PermissionService {
	return &PermissionService{repo: repo}
}

func (s *PermissionService) GetAllPermissions(ctx context.Context) ([]*model.Permission, error) {
	permissions, err := s.repo.GetAllPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("service: get all permissions: %w", err)
	}
	return permissions, nil
}
