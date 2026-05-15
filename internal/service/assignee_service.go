package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/assignee"
	repoPkg "github.com/vega-trello/trello-back/internal/repository"
)

var (
	ErrInvalidUserUUID  = errors.New("user_uuid must be a valid UUID")
	ErrAssigneeNotFound = errors.New("assignee not found")
	ErrAlreadyAssigned  = errors.New("user is already assigned to this task")
	ErrUserNotFound     = errors.New("user not found")
)

type AssigneeService struct {
	repo AssigneeRepository
}

func NewAssigneeService(repo AssigneeRepository) *AssigneeService {
	return &AssigneeService{repo: repo}
}

func toAssigneeDTO(r *repoPkg.AssigneeResponse) *dto.AssigneeResponse {
	return &dto.AssigneeResponse{
		TaskID:     r.TaskID,
		UserUUID:   r.UserUUID.String(),
		AssignedAt: r.AssignedAt,
		User: &dto.UserInfo{
			Username: r.Username,
			UUID:     r.UserUUID.String(),
		},
	}
}

func (s *AssigneeService) GetTaskAssignees(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
) ([]*dto.AssigneeResponse, error) {
	repoAssignees, err := s.repo.FindByTask(ctx, projectUUID, taskID, userUUID)
	if err != nil {
		if errors.Is(err, repoPkg.ErrAccessDenied) {
			return nil, ErrAccessDenied
		}
		if errors.Is(err, repoPkg.ErrTaskNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	dtoAssignees := make([]*dto.AssigneeResponse, len(repoAssignees))
	for i, a := range repoAssignees {
		dtoAssignees[i] = toAssigneeDTO(a)
	}
	return dtoAssignees, nil
}

func (s *AssigneeService) AssignUserToTask(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
	req dto.CreateAssigneeRequest,
) error {
	assigneeUUID, err := uuid.Parse(req.UserUUID)
	if err != nil {
		return ErrInvalidUserUUID
	}

	err = s.repo.Add(ctx, projectUUID, taskID, assigneeUUID, userUUID)
	if err != nil {
		if errors.Is(err, repoPkg.ErrAlreadyAssigned) {
			return ErrAlreadyAssigned
		}
		if errors.Is(err, repoPkg.ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if errors.Is(err, repoPkg.ErrUserNotFound) {
			return ErrUserNotFound
		}
		if errors.Is(err, repoPkg.ErrAccessDenied) {
			return ErrAccessDenied
		}
		return err
	}
	return nil
}

func (s *AssigneeService) RemoveAssignee(
	ctx context.Context,
	projectUUID uuid.UUID,
	taskID int,
	userUUID uuid.UUID,
	assigneeUUID uuid.UUID,
) error {
	err := s.repo.Remove(ctx, projectUUID, taskID, assigneeUUID, userUUID)
	if err != nil {
		if errors.Is(err, repoPkg.ErrAssigneeNotFound) {
			return ErrAssigneeNotFound
		}
		if errors.Is(err, repoPkg.ErrTaskNotFound) {
			return ErrTaskNotFound
		}
		if errors.Is(err, repoPkg.ErrAccessDenied) {
			return ErrAccessDenied
		}
		return err
	}
	return nil
}
