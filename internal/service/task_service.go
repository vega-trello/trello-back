package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/task"
	"github.com/vega-trello/trello-back/internal/model"
)

var (
	ErrInvalidTitle           = errors.New("title must be between 1 and 256 characters")
	ErrInvalidDescriptionTask = errors.New("description must not exceed 2048 characters")
	ErrInvalidDateRange       = errors.New("start_date cannot be after end_date")
	ErrInvalidDateFormat      = errors.New("invalid date format, expected RFC3339")
	ErrTaskNotFound           = errors.New("task not found")
	ErrInvalidColumn          = errors.New("column does not belong to project")
	ErrInvalidStatus          = errors.New("status does not belong to project")
)

type TaskService struct {
	repo TaskRepository
}

func NewTaskService(repo TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

// parseDateString хелпер для конвертации *string (DTO) - *time.Time (репозиторий)
func parseDateString(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, ErrInvalidDateFormat
	}
	return &t, nil
}

// CreateTask создаёт новую задачу
func (s *TaskService) CreateTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	req dto.CreateTaskRequest,
) (*model.TaskDB, error) {
	if len(req.Title) > 256 {
		return nil, ErrInvalidTitle
	}
	if req.Description != "" && len(req.Description) > 2048 {
		return nil, ErrInvalidDescriptionTask
	}

	startDate, err := parseDateString(req.StartDate)
	if err != nil {
		return nil, err
	}
	endDate, err := parseDateString(req.EndDate)
	if err != nil {
		return nil, err
	}
	if startDate != nil && endDate != nil && startDate.After(*endDate) {
		return nil, ErrInvalidDateRange
	}

	task, err := s.repo.Create(ctx, projectUUID, *req.ColumnID, req.StatusID, userUUID, req.Title, req.Description, startDate, endDate)
	if err != nil {
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		if errors.Is(err, ErrInvalidColumn) {
			return nil, ErrInvalidColumn
		}
		if errors.Is(err, ErrInvalidStatus) {
			return nil, ErrInvalidStatus
		}
		return nil, err
	}
	return task, nil
}

// GetProjectTasks возвращает задачи проекта с опциональной фильтрацией по архиву
func (s *TaskService) GetProjectTasks(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	archived *bool,
) ([]*model.TaskDB, error) {
	tasks, err := s.repo.FindByProjectUUID(ctx, projectUUID, userUUID, archived)
	if err != nil {
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}
	return tasks, nil
}

// GetTask возвращает задачу по ID с проверкой доступа
func (s *TaskService) GetTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
) (*model.TaskDB, error) {
	task, err := s.repo.FindByID(ctx, projectUUID, taskID, userUUID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) || errors.Is(err, ErrAccessDenied) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return task, nil
}

// UpdateTask обновляет задачу (частичное обновление)
func (s *TaskService) UpdateTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
	req dto.UpdateTaskRequest,
) (*model.TaskDB, error) {
	if req.Title != nil && len(*req.Title) > 256 {
		return nil, ErrInvalidTitle
	}
	if req.Description != nil && len(*req.Description) > 2048 {
		return nil, ErrInvalidDescriptionTask
	}

	var startDate, endDate *time.Time
	var err error
	if req.StartDate != nil {
		startDate, err = parseDateString(req.StartDate)
		if err != nil {
			return nil, err
		}
	}
	if req.EndDate != nil {
		endDate, err = parseDateString(req.EndDate)
		if err != nil {
			return nil, err
		}
	}
	if startDate != nil && endDate != nil && startDate.After(*endDate) {
		return nil, ErrInvalidDateRange
	}

	updated, err := s.repo.Update(ctx, projectUUID, taskID, userUUID, req.Title, req.Description, startDate, endDate, req.ColumnID, req.StatusID, req.Archived)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		if errors.Is(err, ErrInvalidColumn) {
			return nil, ErrInvalidColumn
		}
		if errors.Is(err, ErrInvalidStatus) {
			return nil, ErrInvalidStatus
		}
		return nil, err
	}
	return updated, nil
}

// DeleteTask удаляет задачу
func (s *TaskService) DeleteTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
) error {
	err := s.repo.Delete(ctx, projectUUID, taskID, userUUID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if errors.Is(err, ErrAccessDenied) {
			return ErrAccessDenied
		}
		return err
	}
	return nil
}

// MoveTask перемещает задачу в другую колонку
func (s *TaskService) MoveTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	columnID int,
	userUUID uuid.UUID,
) error {
	if columnID < 1 {
		return ErrInvalidColumn
	}
	err := s.repo.Move(ctx, projectUUID, taskID, columnID, userUUID)
	if err != nil {
		if errors.Is(err, ErrInvalidColumn) {
			return ErrInvalidColumn
		}
		if errors.Is(err, ErrAccessDenied) {
			return ErrAccessDenied
		}
		if errors.Is(err, ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		return err
	}
	return nil
}

/*
// ArchiveTask архивирует/разархивирует задачу
func (s *TaskService) ArchiveTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
	archive bool,
) error {
	_, err := s.repo.FindByID(ctx, projectUUID, taskID, userUUID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) || errors.Is(err, ErrAccessDenied) {
			return ErrTaskNotFound
		}
		return err
	}

	err = s.repo.Archive(ctx, projectUUID, taskID, userUUID, archive)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if errors.Is(err, ErrAccessDenied) {
			return ErrAccessDenied
		}
		return err
	}
	return nil
}

*/
