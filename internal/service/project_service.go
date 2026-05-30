package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/project"
	"github.com/vega-trello/trello-back/internal/model"
)

var (
	ErrInvalidProjectTitle       = errors.New("project title must be between 1 and 128 characters")
	ErrInvalidDescriptionProject = errors.New("project description must not exceed 512 characters")
	ErrProjectNotFound           = errors.New("project not found")
	ErrAccessDenied              = errors.New("user does not have access to this project")
	ErrProjectHasMembers         = errors.New("cannot delete project with members: remove them first")
	ErrProjectTitleTaken         = errors.New("project with this title already exists")
)

type ProjectService struct {
	repo ProjectRepository
}

func NewProjectService(repo ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

// CreateProject создаёт новый проект и автоматически добавляет создателя как владельца
func (s *ProjectService) CreateProject(
	ctx context.Context,
	userUUID uuid.UUID,
	title string,
	description *string,
) (*model.Project, error) {
	if title == "" || len(title) > 128 {
		return nil, ErrInvalidProjectTitle
	}
	if description != nil && len(*description) > 512 {
		return nil, ErrInvalidDescriptionProject
	}

	req := dto.CreateProjectRequest{
		Title:       title,
		Description: description,
	}

	return s.repo.Create(ctx, userUUID, req)
}

// GetProject возвращает проект по UUID с проверкой доступа
func (s *ProjectService) GetProject(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) (*model.Project, error) {
	isMember, err := s.repo.IsMember(ctx, projectUUID, userUUID)
	if err != nil || !isMember {
		return nil, ErrProjectNotFound
	}

	project, err := s.repo.FindByID(ctx, projectUUID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	return project, nil
}

// GetUserProjects возвращает все проекты, в которых пользователь является участником
func (s *ProjectService) GetUserProjects(
	ctx context.Context,
	userUUID uuid.UUID,
) ([]*model.Project, error) {
	return s.repo.FindByUser(ctx, userUUID)
}

// UpdateProject обновляет проект с проверкой прав доступа
// *string для title/description позволяет реализовать частичное обновление (PATCH)
func (s *ProjectService) UpdateProject(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	title *string,
	description *string,
) (*model.Project, error) {
	if title != nil && (len(*title) == 0 || len(*title) > 128) {
		return nil, ErrInvalidProjectTitle
	}
	if description != nil && len(*description) > 512 {
		return nil, ErrInvalidDescriptionProject
	}

	isMember, err := s.repo.IsMember(ctx, projectUUID, userUUID)
	if err != nil || !isMember {
		return nil, ErrAccessDenied
	}

	updated, err := s.repo.Update(ctx, projectUUID, userUUID, title, description)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}
	return updated, nil
}

// DeleteProject реализует:
// - Если пользователь ВЛАДЕЛЕЦ - удаляет проект полностью
// - Если пользователь УЧАСТНИК (не владелец) - удаляет его из проекта (выход)
// - Если пользователь НЕ имеет доступа - озвращает ошибку
func (s *ProjectService) DeleteProject(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) error {
	isMember, err := s.repo.IsMember(ctx, projectUUID, userUUID)
	if err != nil {
		return fmt.Errorf("service: check member: %w", err)
	}
	if !isMember {
		return ErrProjectNotFound
	}

	isOwner, err := s.repo.IsOwner(ctx, projectUUID, userUUID)
	if err != nil {
		return fmt.Errorf("service: check owner: %w", err)
	}

	if isOwner {
		return s.repo.Delete(ctx, projectUUID)
	}

	return s.repo.RemoveMember(ctx, projectUUID, userUUID)
}
