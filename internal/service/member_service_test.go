//go:build !integration
// +build !integration

package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	dto "github.com/vega-trello/trello-back/internal/dto/member"
	"github.com/vega-trello/trello-back/internal/model"
)

type MockMemberRepository struct {
	mock.Mock
}

func (m *MockMemberRepository) Create(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, roleID int) (*model.ProjectMember, error) {
	args := m.Called(ctx, projectUUID, userUUID, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ProjectMember), args.Error(1)
}

func (m *MockMemberRepository) FindByProjectUUIDWithDetails(ctx context.Context, projectUUID uuid.UUID) ([]*dto.MemberResponse, error) {
	args := m.Called(ctx, projectUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*dto.MemberResponse), args.Error(1)
}

func (m *MockMemberRepository) FindByProjectAndUser(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) (*model.ProjectMember, error) {
	args := m.Called(ctx, projectUUID, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ProjectMember), args.Error(1)
}

func (m *MockMemberRepository) Update(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, roleID int) (*model.ProjectMember, error) {
	args := m.Called(ctx, projectUUID, userUUID, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ProjectMember), args.Error(1)
}

func (m *MockMemberRepository) Delete(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) error {
	args := m.Called(ctx, projectUUID, userUUID)
	return args.Error(0)
}

func TestMemberService_GetProjectMembers_Success(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	expected := []*dto.MemberResponse{
		{Username: "alice", UserUUID: userUUID.String(), RoleID: 1},
		{Username: "bob", UserUUID: uuid.New().String(), RoleID: 2},
	}

	// Проверка доступа вызывающего
	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, userUUID).
		Return(&model.ProjectMember{UserUUID: userUUID}, nil)
	// Получение списка с деталями
	mockRepo.On("FindByProjectUUIDWithDetails", ctx, projectUUID).
		Return(expected, nil)

	members, err := svc.GetProjectMembers(ctx, projectUUID, userUUID)

	assert.NoError(t, err)
	assert.Equal(t, expected, members)
	mockRepo.AssertExpectations(t)
}

func TestMemberService_GetProjectMembers_AccessDenied(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	// Вызывающий не является участником
	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, userUUID).
		Return(nil, ErrMemberNotFound)

	_, err := svc.GetProjectMembers(ctx, projectUUID, userUUID)

	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}

func TestMemberService_GetMember_Success(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	targetUUID := uuid.New()

	expected := &dto.MemberResponse{
		Username: "bob",
		UserUUID: targetUUID.String(),
		RoleID:   2,
	}

	// Проверка доступа вызывающего
	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, userUUID).
		Return(&model.ProjectMember{UserUUID: userUUID}, nil)
	// Проверка, что целевой пользователь — участник
	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, targetUUID).
		Return(&model.ProjectMember{UserUUID: targetUUID}, nil)
	// Получение списка для поиска (временное решение)
	mockRepo.On("FindByProjectUUIDWithDetails", ctx, projectUUID).
		Return([]*dto.MemberResponse{expected}, nil)

	member, err := svc.GetMember(ctx, projectUUID, userUUID, targetUUID)

	assert.NoError(t, err)
	assert.Equal(t, expected, member)
	mockRepo.AssertExpectations(t)
}

func TestMemberService_GetMember_NotFound(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	targetUUID := uuid.New()

	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, userUUID).
		Return(&model.ProjectMember{UserUUID: userUUID}, nil)
	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, targetUUID).
		Return(nil, ErrMemberNotFound)

	_, err := svc.GetMember(ctx, projectUUID, userUUID, targetUUID)

	assert.ErrorIs(t, err, ErrMemberNotFound)
	mockRepo.AssertExpectations(t)
}

func TestMemberService_AddMember_Success(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	targetUUID := uuid.New()
	req := dto.CreateMemberRequest{
		UserUUID: targetUUID.String(),
		RoleID:   2,
	}

	expected := &model.ProjectMember{
		ProjectUUID: projectUUID,
		UserUUID:    targetUUID,
		RoleID:      2,
	}

	// Проверка доступа вызывающего
	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, userUUID).
		Return(&model.ProjectMember{UserUUID: userUUID}, nil)
	// Создание участника
	mockRepo.On("Create", ctx, projectUUID, targetUUID, 2).
		Return(expected, nil)

	member, err := svc.AddMember(ctx, projectUUID, userUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, expected, member)
	mockRepo.AssertExpectations(t)
}

func TestMemberService_AddMember_InvalidUUID(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	req := dto.CreateMemberRequest{UserUUID: "not-a-uuid", RoleID: 2}

	_, err := svc.AddMember(ctx, projectUUID, userUUID, req)

	assert.ErrorIs(t, err, ErrInvalidUUID)
	mockRepo.AssertNotCalled(t, "Create")
}

func TestMemberService_AddMember_AlreadyExists(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	targetUUID := uuid.New()
	req := dto.CreateMemberRequest{UserUUID: targetUUID.String(), RoleID: 2}

	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, userUUID).
		Return(&model.ProjectMember{UserUUID: userUUID}, nil)
	mockRepo.On("Create", ctx, projectUUID, targetUUID, 2).
		Return(nil, ErrMemberAlreadyExists)

	_, err := svc.AddMember(ctx, projectUUID, userUUID, req)

	assert.ErrorIs(t, err, ErrMemberAlreadyExists)
	mockRepo.AssertExpectations(t)
}

func TestMemberService_UpdateMemberRole_Success(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	targetUUID := uuid.New()
	req := dto.UpdateMemberRequest{RoleID: 3}

	expected := &model.ProjectMember{
		ProjectUUID: projectUUID,
		UserUUID:    targetUUID,
		RoleID:      3,
	}

	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, userUUID).
		Return(&model.ProjectMember{UserUUID: userUUID}, nil)
	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, targetUUID).
		Return(&model.ProjectMember{UserUUID: targetUUID}, nil)
	mockRepo.On("Update", ctx, projectUUID, targetUUID, 3).
		Return(expected, nil)

	member, err := svc.UpdateMemberRole(ctx, projectUUID, userUUID, targetUUID, req)

	assert.NoError(t, err)
	assert.Equal(t, expected, member)
	mockRepo.AssertExpectations(t)
}

func TestMemberService_UpdateMemberRole_CannotRemoveLastOwner(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	targetUUID := uuid.New()
	req := dto.UpdateMemberRequest{RoleID: 2}

	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, userUUID).
		Return(&model.ProjectMember{UserUUID: userUUID}, nil)
	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, targetUUID).
		Return(&model.ProjectMember{UserUUID: targetUUID}, nil)
	mockRepo.On("Update", ctx, projectUUID, targetUUID, 2).
		Return(nil, ErrCannotRemoveLastOwner)

	_, err := svc.UpdateMemberRole(ctx, projectUUID, userUUID, targetUUID, req)

	assert.ErrorIs(t, err, ErrCannotRemoveLastOwner)
	mockRepo.AssertExpectations(t)
}

func TestMemberService_RemoveMember_Success(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	targetUUID := uuid.New()

	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, userUUID).
		Return(&model.ProjectMember{UserUUID: userUUID}, nil)
	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, targetUUID).
		Return(&model.ProjectMember{UserUUID: targetUUID}, nil)
	mockRepo.On("Delete", ctx, projectUUID, targetUUID).
		Return(nil)

	err := svc.RemoveMember(ctx, projectUUID, userUUID, targetUUID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMemberService_RemoveMember_CannotRemoveSelf(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	err := svc.RemoveMember(ctx, projectUUID, userUUID, userUUID)

	assert.ErrorIs(t, err, ErrCannotRemoveSelf)
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestMemberService_RemoveMember_CannotRemoveLastOwner(t *testing.T) {
	mockRepo := new(MockMemberRepository)
	svc := NewMemberService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	targetUUID := uuid.New()

	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, userUUID).
		Return(&model.ProjectMember{UserUUID: userUUID}, nil)
	mockRepo.On("FindByProjectAndUser", ctx, projectUUID, targetUUID).
		Return(&model.ProjectMember{UserUUID: targetUUID}, nil)
	mockRepo.On("Delete", ctx, projectUUID, targetUUID).
		Return(ErrCannotRemoveLastOwner)

	err := svc.RemoveMember(ctx, projectUUID, userUUID, targetUUID)

	assert.ErrorIs(t, err, ErrCannotRemoveLastOwner)
	mockRepo.AssertExpectations(t)
}
