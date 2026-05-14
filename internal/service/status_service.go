package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/status"
	"github.com/vega-trello/trello-back/internal/model"
)

var (
	ErrInvalidStatusName    = errors.New("status name must be between 1 and 32 characters")
	ErrStatusAlreadyExists  = errors.New("status with this name already exists in project")
	ErrStatusNotFound       = errors.New("project status not found")
	ErrStatusHasActiveTasks = errors.New("cannot delete status with active tasks")
)

type StatusService struct {
	repo StatusRepository
}

func NewStatusService(repo StatusRepository) *StatusService {
	return &StatusService{repo: repo}
}

// CreateStatus создаёт новый статус в проекте
func (s *StatusService) CreateStatus(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	req dto.CreateStatusRequest,
) (*model.ProjectStatus, error) {
	if req.Name == "" || len(req.Name) > 32 {
		return nil, ErrInvalidStatusName
	}

	status, err := s.repo.Create(ctx, projectUUID, req.Name, userUUID)
	if err != nil {
		if errors.Is(err, ErrStatusAlreadyExists) {
			return nil, ErrStatusAlreadyExists
		}
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}
	return status, nil
}

// GetProjectStatuses возвращает все статусы проекта
func (s *StatusService) GetProjectStatuses(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) ([]*model.ProjectStatus, error) {
	statuses, err := s.repo.FindByProject(ctx, projectUUID, userUUID)
	if err != nil {
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}
	return statuses, nil
}

// GetStatus возвращает статус по ID с проверкой доступа
func (s *StatusService) GetStatus(
	ctx context.Context,
	projectUUID uuid.UUID,
	statusID int,
	userUUID uuid.UUID,
) (*model.ProjectStatus, error) {
	status, err := s.repo.FindByID(ctx, projectUUID, statusID, userUUID)
	if err != nil {
		if errors.Is(err, ErrStatusNotFound) || errors.Is(err, ErrAccessDenied) {
			return nil, ErrStatusNotFound
		}
		return nil, err
	}
	return status, nil
}

// UpdateStatus обновляет имя статуса
func (s *StatusService) UpdateStatus(
	ctx context.Context,
	projectUUID uuid.UUID,
	statusID int,
	userUUID uuid.UUID,
	req dto.UpdateStatusRequest,
) (*model.ProjectStatus, error) {
	if req.Name == "" || len(req.Name) > 32 {
		return nil, ErrInvalidStatusName
	}

	updated, err := s.repo.Update(ctx, projectUUID, statusID, req.Name, userUUID)
	if err != nil {
		if errors.Is(err, ErrStatusAlreadyExists) {
			return nil, ErrStatusAlreadyExists
		}
		if errors.Is(err, ErrStatusNotFound) {
			return nil, ErrStatusNotFound
		}
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}
	return updated, nil
}

// DeleteStatus удаляет статус (только если нет активных задач)
func (s *StatusService) DeleteStatus(
	ctx context.Context,
	projectUUID uuid.UUID,
	statusID int,
	userUUID uuid.UUID,
) error {
	err := s.repo.Delete(ctx, projectUUID, statusID, userUUID)
	if err != nil {
		if errors.Is(err, ErrStatusHasActiveTasks) {
			return ErrStatusHasActiveTasks
		}
		if errors.Is(err, ErrStatusNotFound) {
			return ErrStatusNotFound
		}
		if errors.Is(err, ErrAccessDenied) {
			return ErrAccessDenied
		}
		return err
	}
	return nil
}
