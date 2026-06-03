package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/column"
	"github.com/vega-trello/trello-back/internal/model"
)

var (
	ErrInvalidColumnName = errors.New("column name must be between 1 and 64 characters")
	ErrInvalidPosition   = errors.New("position must be non-negative")
	ErrInvalidDirection  = errors.New("direction must be 'left' or 'right'")
	ErrColumnNotFound    = errors.New("column not found")
	ErrColumnHasTasks    = errors.New("cannot delete column with tasks: remove them first")
)

type ColumnService struct {
	repo ColumnRepository
}

func NewColumnService(repo ColumnRepository) *ColumnService {
	return &ColumnService{repo: repo}
}

// CreateColumn создаёт новую колонку в проекте
func (s *ColumnService) CreateColumn(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	req dto.CreateColumnRequest,
) (*model.Column, error) {
	if req.Name == "" || len(req.Name) > 64 {
		return nil, ErrInvalidColumnName
	}
	if req.Position != nil && *req.Position < 0 {
		return nil, ErrInvalidPosition
	}

	column, err := s.repo.Create(ctx, projectUUID, userUUID, req.Name, req.Position, nil)
	if err != nil {
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}
	return column, nil
}

// GetProjectColumns возвращает все колонки проекта
func (s *ColumnService) GetProjectColumns(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) ([]*model.Column, error) {
	columns, err := s.repo.FindByProjectUUID(ctx, projectUUID, userUUID)
	if err != nil {
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}
	return columns, nil
}

// GetColumn возвращает колонку по ID
func (s *ColumnService) GetColumn(
	ctx context.Context,
	columnID int,
	userUUID uuid.UUID,
) (*model.Column, error) {
	column, err := s.repo.FindByID(ctx, columnID, userUUID)
	if err != nil {
		if errors.Is(err, ErrColumnNotFound) || errors.Is(err, ErrAccessDenied) {
			return nil, ErrColumnNotFound
		}
		return nil, err
	}
	return column, nil
}

func (s *ColumnService) UpdateColumn(
	ctx context.Context,
	columnID int,
	userUUID uuid.UUID,
	req dto.UpdateColumnRequest,
) (*model.Column, error) {
	if req.Name == "" || len(req.Name) > 64 {
		return nil, ErrInvalidColumnName
	}
	if req.Position != nil && *req.Position < 0 {
		return nil, ErrInvalidPosition
	}

	colorVal, shouldUpdate := req.GetColor()

	if shouldUpdate && colorVal != nil && !isValidHexColor(*colorVal) {
		return nil, errors.New("color must be a valid HEX string (#RRGGBB or #RRGGBBAA)")
	}

	updated, err := s.repo.Update(ctx, columnID, userUUID, req.Name, req.Position, colorVal, shouldUpdate)
	if err != nil {
		if errors.Is(err, ErrColumnNotFound) {
			return nil, ErrColumnNotFound
		}
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}
	return updated, nil
}

func (s *ColumnService) DeleteColumn(
	ctx context.Context,
	columnID int,
	userUUID uuid.UUID,
) error {
	err := s.repo.Delete(ctx, columnID, userUUID)
	if err != nil {
		if errors.Is(err, ErrColumnHasTasks) {
			return ErrColumnHasTasks
		}
		if errors.Is(err, ErrColumnNotFound) {
			return ErrColumnNotFound
		}
		if errors.Is(err, ErrAccessDenied) {
			return ErrAccessDenied
		}
		return err
	}
	return nil
}

// MoveColumn перемещает колонку влево/вправо
func (s *ColumnService) MoveColumn(
	ctx context.Context,
	columnID int,
	userUUID uuid.UUID,
	direction string,
) (*model.Column, error) {
	if direction != "left" && direction != "right" {
		return nil, ErrInvalidDirection
	}

	updated, err := s.repo.Move(ctx, columnID, userUUID, direction)
	if err != nil {
		if errors.Is(err, ErrColumnNotFound) {
			return nil, ErrColumnNotFound
		}
		if errors.Is(err, ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}
	return updated, nil
}

func isValidHexColor(s string) bool {
	if len(s) != 7 && len(s) != 9 {
		return false
	}
	if s[0] != '#' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
