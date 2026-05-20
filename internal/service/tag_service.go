package service

import (
	"context"
	"errors"
	"regexp"

	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/tag"
	"github.com/vega-trello/trello-back/internal/model"
)

var (
	ErrInvalidTagName     = errors.New("tag name must be between 1 and 32 characters")
	ErrInvalidColorFormat = errors.New("color must match pattern #RRGGBB (e.g., #FF0000)")
	ErrTagAlreadyExists   = errors.New("tag with this name already exists in project")
	ErrTagNotFound        = errors.New("tag not found")
	ErrTagNotInProject    = errors.New("tag does not belong to this project")
	ErrTaskNotInProject   = errors.New("task does not belong to this project")
)

// hexColorRegex для валидации в сервисе (дублирует DTO для надёжности)
var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type TagService struct {
	repo TagRepository
}

func NewTagService(repo TagRepository) *TagService {
	return &TagService{repo: repo}
}

// validateTagInput общая валидация для Create/Update
func validateTagInput(name, color string) error {
	if name == "" || len(name) > 32 {
		return ErrInvalidTagName
	}
	if !hexColorRegex.MatchString(color) {
		return ErrInvalidColorFormat
	}
	return nil
}

// GetProjectTags возвращает все теги проекта
func (s *TagService) GetProjectTags(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) ([]*model.Tag, error) {
	tags, err := s.repo.FindByProjectUUID(ctx, projectUUID, userUUID)
	if err != nil {
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}
	return tags, nil
}

// GetTaskTags возвращает теги, привязанные к конкретной задаче
func (s *TagService) GetTaskTags(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
) ([]*model.Tag, error) {
	tags, err := s.repo.FindByTask(ctx, projectUUID, taskID, userUUID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) || errors.Is(err, ErrAccessDenied) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return tags, nil
}

// CreateTag создаёт новый тег в проекте
func (s *TagService) CreateTag(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	req dto.CreateTagRequest,
) (*model.Tag, error) {
	if err := validateTagInput(req.Name, req.Color); err != nil {
		return nil, err
	}

	tag, err := s.repo.Create(ctx, projectUUID, userUUID, req.Name, req.Color)
	if err != nil {
		// Репозиторий может вернуть ошибку уникальности (зависит от реализации)
		// if errors.Is(err, repository.ErrTagAlreadyExists) {
		//     return nil, ErrTagAlreadyExists
		// }
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}
	return tag, nil
}

// UpdateTag обновляет имя и/или цвет тега
func (s *TagService) UpdateTag(
	ctx context.Context,
	tagID int,
	userUUID uuid.UUID,
	req dto.UpdateTagRequest,
) (*model.Tag, error) {
	if err := validateTagInput(req.Name, req.Color); err != nil {
		return nil, err
	}

	tag, err := s.repo.Update(ctx, tagID, userUUID, req.Name, req.Color)
	if err != nil {
		if errors.Is(err, ErrTagNotFound) {
			return nil, ErrTagNotFound
		}
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		// if errors.Is(err, repository.ErrTagAlreadyExists) {
		//     return nil, ErrTagAlreadyExists
		// }
		return nil, err
	}
	return tag, nil
}

// DeleteTag удаляет тег (отвязывает от всех задач автоматически через CASCADE или логику БД)
func (s *TagService) DeleteTag(
	ctx context.Context,
	tagID int,
	userUUID uuid.UUID,
) error {
	err := s.repo.Delete(ctx, tagID, userUUID)
	if err != nil {
		if errors.Is(err, ErrTagNotFound) {
			return ErrTagNotFound
		}
		if errors.Is(err, ErrAccessDenied) {
			return ErrAccessDenied
		}
		return err
	}
	return nil
}

// AddTagToTask привязывает тег к задаче
func (s *TagService) AddTagToTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	taskID int,
	tagID int,
) error {
	if taskID < 1 || tagID < 1 {
		return ErrInvalidTagName
	}

	err := s.repo.AddToTask(ctx, projectUUID, userUUID, taskID, tagID)
	if err != nil {
		if errors.Is(err, ErrTagNotFound) {
			return ErrTagNotFound
		}
		if errors.Is(err, ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if errors.Is(err, ErrAccessDenied) {
			return ErrAccessDenied
		}
		if err.Error() == "tag does not belong to this project" {
			return ErrTagNotInProject
		}
		if err.Error() == "task does not belong to this project" {
			return ErrTaskNotInProject
		}
		return err
	}
	return nil
}

// RemoveTagFromTask отвязывает тег от задачи
func (s *TagService) RemoveTagFromTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	taskID int,
	tagID int,
) error {
	if taskID < 1 || tagID < 1 {
		return ErrInvalidTagName
	}

	err := s.repo.RemoveFromTask(ctx, projectUUID, userUUID, taskID, tagID)
	if err != nil {
		if errors.Is(err, ErrTagNotFound) {
			return ErrTagNotFound
		}
		if errors.Is(err, ErrAccessDenied) {
			return ErrAccessDenied
		}
		return err
	}
	return nil
}
