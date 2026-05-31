package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/member"
	"github.com/vega-trello/trello-back/internal/model"
)

var (
	ErrMemberNotFound        = errors.New("member not found")
	ErrMemberAlreadyExists   = errors.New("user is already a member of this project")
	ErrCannotRemoveLastOwner = errors.New("cannot remove the last owner of the project")
	ErrCannotRemoveSelf      = errors.New("cannot remove yourself from the project")
	ErrInvalidRole           = errors.New("invalid role ID")
	ErrInvalidUUID           = errors.New("invalid UUID format")
)

type MemberService struct {
	repo MemberRepository
}

func NewMemberService(repo MemberRepository) *MemberService {
	return &MemberService{repo: repo}
}

// GetProjectMembers возвращает всех участников проекта с деталями
func (s *MemberService) GetProjectMembers(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
) ([]*dto.MemberResponse, error) {
	_, err := s.repo.FindByProjectAndUser(ctx, projectUUID, userUUID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}

	members, err := s.repo.FindByProjectUUIDWithDetails(ctx, projectUUID)
	if err != nil {
		return nil, err
	}
	return members, nil
}

// GetMember возвращает детали конкретного участника
func (s *MemberService) GetMember(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	targetUserUUID uuid.UUID,
) (*dto.MemberResponse, error) {
	_, err := s.repo.FindByProjectAndUser(ctx, projectUUID, userUUID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}

	_, err = s.repo.FindByProjectAndUser(ctx, projectUUID, targetUserUUID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, err
	}

	members, err := s.repo.FindByProjectUUIDWithDetails(ctx, projectUUID)
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		if m.UUID == targetUserUUID.String() {
			return m, nil
		}
	}
	return nil, ErrMemberNotFound
}

// AddMember добавляет пользователя в проект
func (s *MemberService) AddMember(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	req dto.CreateMemberRequest,
) (*model.ProjectMember, error) {
	if req.RoleID < 1 {
		return nil, ErrInvalidRole
	}

	targetUUID, err := uuid.Parse(req.UserUUID)
	if err != nil {
		return nil, ErrInvalidUUID
	}

	// нельзя добавить самого себя - сделать позже
	// if userUUID == targetUUID {
	//     return nil, errors.New("cannot add yourself")
	// }

	_, err = s.repo.FindByProjectAndUser(ctx, projectUUID, userUUID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}

	member, err := s.repo.Create(ctx, projectUUID, targetUUID, req.RoleID)
	if err != nil {
		if errors.Is(err, ErrMemberAlreadyExists) {
			return nil, ErrMemberAlreadyExists
		}
		// Ошибка внешнего ключа (пользователь не найден) можно обработать отдельно позже
		// if strings.Contains(err.Error(), "foreign key") {
		//     return nil, ErrUserNotFound
		// }
		return nil, err
	}
	return member, nil
}

// UpdateMemberRole изменяет роль участника
func (s *MemberService) UpdateMemberRole(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	targetUserUUID uuid.UUID,
	req dto.UpdateMemberRequest,
) (*model.ProjectMember, error) {
	if req.RoleID < 1 {
		return nil, ErrInvalidRole
	}

	_, err := s.repo.FindByProjectAndUser(ctx, projectUUID, userUUID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return nil, ErrAccessDenied
		}
		return nil, err
	}

	_, err = s.repo.FindByProjectAndUser(ctx, projectUUID, targetUserUUID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, err
	}

	member, err := s.repo.Update(ctx, projectUUID, targetUserUUID, req.RoleID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return nil, ErrMemberNotFound
		}
		if errors.Is(err, ErrCannotRemoveLastOwner) {
			return nil, ErrCannotRemoveLastOwner
		}
		return nil, err
	}
	return member, nil
}

// RemoveMember удаляет участника из проекта
func (s *MemberService) RemoveMember(
	ctx context.Context,
	projectUUID uuid.UUID,
	userUUID uuid.UUID,
	targetUserUUID uuid.UUID,
) error {
	if userUUID == targetUserUUID {
		return ErrCannotRemoveSelf
	}

	_, err := s.repo.FindByProjectAndUser(ctx, projectUUID, userUUID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return ErrAccessDenied
		}
		return err
	}

	_, err = s.repo.FindByProjectAndUser(ctx, projectUUID, targetUserUUID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return ErrMemberNotFound
		}
		return err
	}

	err = s.repo.Delete(ctx, projectUUID, targetUserUUID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return ErrMemberNotFound
		}
		if errors.Is(err, ErrCannotRemoveLastOwner) {
			return ErrCannotRemoveLastOwner
		}
		return err
	}
	return nil
}
