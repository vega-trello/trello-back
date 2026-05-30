//go:build !integration
// +build !integration

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	dto "github.com/vega-trello/trello-back/internal/dto/project"
	"github.com/vega-trello/trello-back/internal/model"
)

type MockProjectRepository struct {
	mock.Mock
}

func (m *MockProjectRepository) Create(ctx context.Context, userUUID uuid.UUID, req dto.CreateProjectRequest) (*model.Project, error) {
	args := m.Called(ctx, userUUID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectRepository) FindByUser(ctx context.Context, userUUID uuid.UUID) ([]*model.Project, error) {
	args := m.Called(ctx, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Project), args.Error(1)
}

func (m *MockProjectRepository) Update(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, title *string, description *string) (*model.Project, error) {
	args := m.Called(ctx, projectUUID, userUUID, title, description)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectRepository) Delete(ctx context.Context, projectUUID uuid.UUID) error {
	args := m.Called(ctx, projectUUID)
	return args.Error(0)
}

func (m *MockProjectRepository) IsMember(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) (bool, error) {
	args := m.Called(ctx, projectUUID, userUUID)
	return args.Bool(0), args.Error(1)
}

func (m *MockProjectRepository) IsOwner(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) (bool, error) {
	args := m.Called(ctx, projectUUID, userUUID)
	return args.Bool(0), args.Error(1)
}

func (m *MockProjectRepository) RemoveMember(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) error {
	args := m.Called(ctx, projectUUID, userUUID)
	return args.Error(0)
}

func strPtr(s string) *string {
	return &s
}

func TestProjectService_CreateProject_Success(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	title := "New Project"
	desc := "Description"

	expectedProject := &model.Project{
		UUID:        uuid.New(),
		Title:       title,
		Description: &desc,
	}

	mockRepo.On("Create", ctx, userUUID, dto.CreateProjectRequest{
		Title:       title,
		Description: &desc,
	}).Return(expectedProject, nil)

	project, err := svc.CreateProject(ctx, userUUID, title, &desc)

	assert.NoError(t, err)
	assert.Equal(t, expectedProject, project)
	mockRepo.AssertExpectations(t)
}

func TestProjectService_CreateProject_InvalidTitle(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	_, err := svc.CreateProject(ctx, userUUID, "", nil)
	assert.ErrorIs(t, err, ErrInvalidProjectTitle)

	_, err = svc.CreateProject(ctx, userUUID, string(make([]byte, 129)), nil)
	assert.ErrorIs(t, err, ErrInvalidProjectTitle)

	mockRepo.AssertNotCalled(t, "Create")
}

func TestProjectService_CreateProject_InvalidDescription(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()
	longDesc := string(make([]byte, 513)) // 513 символа

	_, err := svc.CreateProject(ctx, userUUID, "Valid", &longDesc)
	assert.ErrorIs(t, err, ErrInvalidDescriptionProject)

	mockRepo.AssertNotCalled(t, "Create")
}

func TestProjectService_GetProject_Success(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	expectedProject := &model.Project{
		UUID:  projectUUID,
		Title: "Test Project",
	}

	// Проверка прав + получение проекта
	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(true, nil)
	mockRepo.On("FindByID", ctx, projectUUID).Return(expectedProject, nil)

	project, err := svc.GetProject(ctx, projectUUID, userUUID)

	assert.NoError(t, err)
	assert.Equal(t, expectedProject, project)
	mockRepo.AssertExpectations(t)
}

func TestProjectService_GetProject_AccessDenied(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	// Пользователь не является участником
	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(false, nil)

	_, err := svc.GetProject(ctx, projectUUID, userUUID)

	assert.ErrorIs(t, err, ErrProjectNotFound) // Маппинг в ErrProjectNotFound для безопасности
	mockRepo.AssertExpectations(t)
}

func TestProjectService_GetProject_NotFound(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(true, nil)
	mockRepo.On("FindByID", ctx, projectUUID).Return(nil, ErrProjectNotFound)

	_, err := svc.GetProject(ctx, projectUUID, userUUID)

	assert.ErrorIs(t, err, ErrProjectNotFound)
	mockRepo.AssertExpectations(t)
}

func TestProjectService_GetUserProjects_Success(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	userUUID := uuid.New()

	expectedProjects := []*model.Project{
		{UUID: uuid.New(), Title: "Project 1"},
		{UUID: uuid.New(), Title: "Project 2"},
	}

	mockRepo.On("FindByUser", ctx, userUUID).Return(expectedProjects, nil)

	projects, err := svc.GetUserProjects(ctx, userUUID)

	assert.NoError(t, err)
	assert.Equal(t, expectedProjects, projects)
	mockRepo.AssertExpectations(t)
}

func TestProjectService_UpdateProject_Success(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	newTitle := "Updated"
	newDesc := "New description"

	expectedProject := &model.Project{
		UUID:        projectUUID,
		Title:       newTitle,
		Description: &newDesc,
	}

	// Проверка прав + обновление
	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(true, nil)
	mockRepo.On("Update", ctx, projectUUID, userUUID, strPtr(newTitle), strPtr(newDesc)).
		Return(expectedProject, nil)

	project, err := svc.UpdateProject(ctx, projectUUID, userUUID, strPtr(newTitle), strPtr(newDesc))

	assert.NoError(t, err)
	assert.Equal(t, expectedProject, project)
	mockRepo.AssertExpectations(t)
}

func TestProjectService_UpdateProject_PartialUpdate_TitleOnly(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	newTitle := "Only Title"

	expectedProject := &model.Project{
		UUID:  projectUUID,
		Title: newTitle,
		// Description не меняется (nil)
	}

	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(true, nil)
	mockRepo.On("Update", ctx, projectUUID, userUUID, strPtr(newTitle), (*string)(nil)).
		Return(expectedProject, nil)

	project, err := svc.UpdateProject(ctx, projectUUID, userUUID, strPtr(newTitle), nil)

	assert.NoError(t, err)
	assert.Equal(t, newTitle, project.Title)
	mockRepo.AssertExpectations(t)
}

func TestProjectService_UpdateProject_PartialUpdate_DescriptionOnly(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	newDesc := "Only Desc"

	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(true, nil)
	mockRepo.On("Update", ctx, projectUUID, userUUID, (*string)(nil), strPtr(newDesc)).
		Return(&model.Project{UUID: projectUUID, Description: &newDesc}, nil)

	project, err := svc.UpdateProject(ctx, projectUUID, userUUID, nil, strPtr(newDesc))

	assert.NoError(t, err)
	assert.Equal(t, newDesc, *project.Description)
	mockRepo.AssertExpectations(t)
}

func TestProjectService_UpdateProject_InvalidTitle(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	// Пустой заголовок (если передан)
	_, err := svc.UpdateProject(ctx, projectUUID, userUUID, strPtr(""), nil)
	assert.ErrorIs(t, err, ErrInvalidProjectTitle)

	// Слишком длинный заголовок
	_, err = svc.UpdateProject(ctx, projectUUID, userUUID, strPtr(string(make([]byte, 129))), nil)
	assert.ErrorIs(t, err, ErrInvalidProjectTitle)

	mockRepo.AssertNotCalled(t, "Update")
}

func TestProjectService_UpdateProject_InvalidDescription(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()
	longDesc := string(make([]byte, 513))

	_, err := svc.UpdateProject(ctx, projectUUID, userUUID, nil, &longDesc)
	assert.ErrorIs(t, err, ErrInvalidDescriptionProject)

	mockRepo.AssertNotCalled(t, "Update")
}

func TestProjectService_UpdateProject_AccessDenied(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(false, nil)

	_, err := svc.UpdateProject(ctx, projectUUID, userUUID, strPtr("New"), nil)

	assert.ErrorIs(t, err, ErrAccessDenied)
	mockRepo.AssertExpectations(t)
}

func TestProjectService_DeleteProject_OwnerDeletesProject(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	// Пользователь — владелец проекта
	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(true, nil)
	mockRepo.On("IsOwner", ctx, projectUUID, userUUID).Return(true, nil)
	mockRepo.On("Delete", ctx, projectUUID).Return(nil)

	err := svc.DeleteProject(ctx, projectUUID, userUUID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	mockRepo.AssertNotCalled(t, "RemoveMember")
}

func TestProjectService_DeleteProject_MemberLeavesProject(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(true, nil)
	mockRepo.On("IsOwner", ctx, projectUUID, userUUID).Return(false, nil)
	mockRepo.On("RemoveMember", ctx, projectUUID, userUUID).Return(nil)

	err := svc.DeleteProject(ctx, projectUUID, userUUID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)

	mockRepo.AssertNotCalled(t, "Delete")
}

func TestProjectService_DeleteProject_NoAccess(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(false, nil)

	err := svc.DeleteProject(ctx, projectUUID, userUUID)

	assert.ErrorIs(t, err, ErrProjectNotFound)
	mockRepo.AssertExpectations(t)

	mockRepo.AssertNotCalled(t, "IsOwner")
	mockRepo.AssertNotCalled(t, "Delete")
	mockRepo.AssertNotCalled(t, "RemoveMember")
}

func TestProjectService_DeleteProject_CheckMemberError(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	dbErr := errors.New("database connection failed")
	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(false, dbErr)

	err := svc.DeleteProject(ctx, projectUUID, userUUID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service: check member")
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "IsOwner")
	mockRepo.AssertNotCalled(t, "Delete")
	mockRepo.AssertNotCalled(t, "RemoveMember")
}

func TestProjectService_DeleteProject_CheckOwnerError(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(true, nil)

	dbErr := errors.New("database error")
	mockRepo.On("IsOwner", ctx, projectUUID, userUUID).Return(false, dbErr)

	err := svc.DeleteProject(ctx, projectUUID, userUUID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service: check owner")
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Delete")
	mockRepo.AssertNotCalled(t, "RemoveMember")
}

func TestProjectService_DeleteProject_DeleteError(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(true, nil)
	mockRepo.On("IsOwner", ctx, projectUUID, userUUID).Return(true, nil)

	deleteErr := ErrProjectNotFound
	mockRepo.On("Delete", ctx, projectUUID).Return(deleteErr)

	err := svc.DeleteProject(ctx, projectUUID, userUUID)

	assert.ErrorIs(t, err, deleteErr)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "RemoveMember")
}

func TestProjectService_DeleteProject_RemoveMemberError(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)

	ctx := context.Background()
	projectUUID := uuid.New()
	userUUID := uuid.New()

	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(true, nil)
	mockRepo.On("IsOwner", ctx, projectUUID, userUUID).Return(false, nil)

	removeErr := errors.New("failed to remove member")
	mockRepo.On("RemoveMember", ctx, projectUUID, userUUID).Return(removeErr)

	err := svc.DeleteProject(ctx, projectUUID, userUUID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove member")
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestProjectService_FullWorkflow_Owner(t *testing.T) {
	mockRepo := new(MockProjectRepository)
	svc := NewProjectService(mockRepo)
	ctx := context.Background()

	userUUID := uuid.New()
	projectUUID := uuid.New()

	mockRepo.On("Create", ctx, userUUID, mock.Anything).Return(&model.Project{UUID: projectUUID, Title: "Test"}, nil)
	project, err := svc.CreateProject(ctx, userUUID, "Test", nil)
	assert.NoError(t, err)
	assert.Equal(t, "Test", project.Title)

	mockRepo.On("IsMember", ctx, projectUUID, userUUID).Return(true, nil)
	mockRepo.On("Update", ctx, projectUUID, userUUID, strPtr("Updated"), (*string)(nil)).
		Return(&model.Project{UUID: projectUUID, Title: "Updated"}, nil)
	updated, err := svc.UpdateProject(ctx, projectUUID, userUUID, strPtr("Updated"), nil)
	assert.NoError(t, err)
	assert.Equal(t, "Updated", updated.Title)

	mockRepo.On("IsOwner", ctx, projectUUID, userUUID).Return(true, nil)
	mockRepo.On("Delete", ctx, projectUUID).Return(nil)
	err = svc.DeleteProject(ctx, projectUUID, userUUID)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}
