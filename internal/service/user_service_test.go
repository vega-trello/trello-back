//go:build !integration
// +build !integration

package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vega-trello/trello-back/internal/model"
	"github.com/vega-trello/trello-back/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) RegisterPasswordUser(ctx context.Context, username string, passwordHash []byte) (*model.User, error) {
	args := m.Called(ctx, username, passwordHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) FindUserByUsername(ctx context.Context, username string) (*model.User, []byte, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*model.User), args.Get(1).([]byte), args.Error(2)
}

func (m *MockUserRepository) FindOrCreateUserBySSO(
	ctx context.Context,
	provider string,
	externalID string,
	username string,
	metadata json.RawMessage,
) (*model.User, error) {
	args := m.Called(ctx, provider, externalID, username, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) FindUserByUUID(ctx context.Context, userUUID uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetSelfUser(ctx context.Context, userUUID uuid.UUID) (*model.SelfUser, error) {
	args := m.Called(ctx, userUUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SelfUser), args.Error(1)
}

func (m *MockUserRepository) UpdateSelfUser(
	ctx context.Context,
	userUUID uuid.UUID,
	oldPassword string,
	newUsername string,
	newPassword string,
) (*model.SelfUser, error) {
	args := m.Called(ctx, userUUID, oldPassword, newUsername, newPassword)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SelfUser), args.Error(1)
}

func (m *MockUserRepository) Logout(ctx context.Context, userUUID uuid.UUID) error {
	args := m.Called(ctx, userUUID)
	return args.Error(0)
}

func hashPassword(t *testing.T, password string) []byte {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)
	return hash
}

func TestUserService_Register_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	username := "newuser"
	password := "StrongPass123"

	testUUID := uuid.New()
	expectedUser := &model.User{
		UUID:     testUUID,
		Username: username,
		UserType: "manual",
	}

	mockRepo.On("RegisterPasswordUser", ctx, username, mock.AnythingOfType("[]uint8")).
		Return(expectedUser, nil)

	user, err := svc.Register(ctx, username, password)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser.UUID, user.UUID)
	assert.Equal(t, expectedUser.Username, user.Username)
	assert.Equal(t, "manual", user.UserType)
	mockRepo.AssertExpectations(t)
}

func TestUserService_Register_PasswordTooShort(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()

	_, err := svc.Register(ctx, "user", "short")

	assert.ErrorIs(t, err, ErrPasswordTooShort)
	mockRepo.AssertNotCalled(t, "RegisterPasswordUser")
}

func TestUserService_Register_UsernameTaken(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	username := "taken"
	password := "StrongPass123"

	mockRepo.On("RegisterPasswordUser", ctx, username, mock.Anything).
		Return(nil, repository.ErrUserAlreadyExists)

	_, err := svc.Register(ctx, username, password)

	assert.ErrorIs(t, err, ErrUserAlreadyExists)
	mockRepo.AssertExpectations(t)
}

func TestUserService_Login_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	username := "loginuser"
	password := "CorrectPass1"
	hash := hashPassword(t, password)

	testUUID := uuid.New()
	expectedUser := &model.User{
		UUID:     testUUID,
		Username: username,
		UserType: "manual",
	}

	mockRepo.On("FindUserByUsername", ctx, username).
		Return(expectedUser, hash, nil)

	result, err := svc.Login(ctx, username, password)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser.UUID, result.UUID)
	assert.Equal(t, expectedUser.Username, result.Username)
	mockRepo.AssertExpectations(t)
}

func TestUserService_Login_WrongPassword(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	username := "user"
	hash := hashPassword(t, "RealPass1")

	mockRepo.On("FindUserByUsername", ctx, username).
		Return(&model.User{UserType: "manual"}, hash, nil)

	_, err := svc.Login(ctx, username, "WrongPass")

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	mockRepo.AssertExpectations(t)
}

func TestUserService_Login_UserNotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	username := "ghost"

	mockRepo.On("FindUserByUsername", ctx, username).
		Return(nil, nil, repository.ErrUserNotFound)

	_, err := svc.Login(ctx, username, "pass")

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	mockRepo.AssertExpectations(t)
}

func TestUserService_LoginBySSO_NewUser_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	provider := "vega"
	externalID := "530"
	username := "petrov"
	metadata := json.RawMessage(`{"fir":"Иван","grn":"КМБО-02-22"}`)

	testUUID := uuid.New()
	expectedUser := &model.User{
		UUID:     testUUID,
		Username: username,
		UserType: "sso",
	}

	mockRepo.On("FindOrCreateUserBySSO", ctx, provider, externalID, username, metadata).
		Return(expectedUser, nil)

	user, err := svc.LoginBySSO(ctx, provider, externalID, username, metadata)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser.UUID, user.UUID)
	assert.Equal(t, "sso", user.UserType)
	mockRepo.AssertExpectations(t)
}

func TestUserService_LoginBySSO_ExistingUser_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	provider := "vega"
	externalID := "530"
	username := "petrov"
	metadata := json.RawMessage(`{}`)

	testUUID := uuid.New()
	existingUser := &model.User{
		UUID:     testUUID,
		Username: "petrov",
		UserType: "sso",
	}

	mockRepo.On("FindOrCreateUserBySSO", ctx, provider, externalID, username, metadata).
		Return(existingUser, nil)

	user, err := svc.LoginBySSO(ctx, provider, externalID, username, metadata)

	assert.NoError(t, err)
	assert.Equal(t, existingUser.UUID, user.UUID)
	assert.Equal(t, "sso", user.UserType)
	mockRepo.AssertExpectations(t)
}

func TestUserService_LoginBySSO_Validation_EmptyProvider(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()

	_, err := svc.LoginBySSO(ctx, "", "530", "user", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider, externalID and username are required")
	mockRepo.AssertNotCalled(t, "FindOrCreateUserBySSO")
}

func TestUserService_LoginBySSO_Validation_EmptyExternalID(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()

	_, err := svc.LoginBySSO(ctx, "vega", "", "user", nil)

	assert.Error(t, err)
	mockRepo.AssertNotCalled(t, "FindOrCreateUserBySSO")
}

func TestUserService_LoginBySSO_UsernameConflict(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	provider := "vega"
	externalID := "999"
	username := "conflict"
	metadata := json.RawMessage(`{}`)

	mockRepo.On("FindOrCreateUserBySSO", ctx, provider, externalID, username, metadata).
		Return(nil, repository.ErrUserAlreadyExists)

	_, err := svc.LoginBySSO(ctx, provider, externalID, username, metadata)

	assert.ErrorIs(t, err, ErrUserAlreadyExists)
	mockRepo.AssertExpectations(t)
}

func TestUserService_LoginBySSO_RepoError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	provider := "vega"
	externalID := "530"
	username := "user"
	metadata := json.RawMessage(`{}`)
	repoErr := errors.New("database connection failed")

	mockRepo.On("FindOrCreateUserBySSO", ctx, provider, externalID, username, metadata).
		Return(nil, repoErr)

	_, err := svc.LoginBySSO(ctx, provider, externalID, username, metadata)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service: SSO login")
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetSelfProfile_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()

	expectedSelf := &model.SelfUser{
		User: model.User{
			UUID:     testUUID,
			Username: "profile",
			UserType: "manual",
		},
	}

	mockRepo.On("GetSelfUser", ctx, testUUID).Return(expectedSelf, nil)

	result, err := svc.GetSelfProfile(ctx, testUUID)

	assert.NoError(t, err)
	assert.Equal(t, expectedSelf.UUID, result.UUID)
	assert.Equal(t, "manual", result.UserType)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetSelfProfile_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()

	mockRepo.On("GetSelfUser", ctx, testUUID).Return(nil, repository.ErrUserNotFound)

	_, err := svc.GetSelfProfile(ctx, testUUID)

	assert.ErrorIs(t, err, repository.ErrUserNotFound)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetOtherUserProfile_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	targetUUID := uuid.New()

	// Репозиторий возвращает полную модель (возможно, с чувствительными полями)
	fullUser := &model.User{
		UUID:     targetUUID,
		Username: "otheruser",
		UserType: "manual",
	}

	mockRepo.On("FindUserByUUID", ctx, targetUUID).Return(fullUser, nil)

	result, err := svc.GetOtherUserProfile(ctx, targetUUID)

	assert.NoError(t, err)
	// Сервис возвращает только публичные поля
	assert.Equal(t, targetUUID, result.UUID)
	assert.Equal(t, "otheruser", result.Username)
	assert.Equal(t, "manual", result.UserType)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetOtherUserProfile_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	targetUUID := uuid.New()

	mockRepo.On("FindUserByUUID", ctx, targetUUID).Return(nil, repository.ErrUserNotFound)

	_, err := svc.GetOtherUserProfile(ctx, targetUUID)

	assert.ErrorIs(t, err, repository.ErrUserNotFound)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetOtherUserProfile_RepoError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	targetUUID := uuid.New()
	repoErr := errors.New("database error")

	mockRepo.On("FindUserByUUID", ctx, targetUUID).Return(nil, repoErr)

	_, err := svc.GetOtherUserProfile(ctx, targetUUID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service: get other user profile")
	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdateSelfProfile_Success_ChangeUsername(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()
	oldPass := "oldpass"
	newUsername := "newname"
	newPass := ""

	updatedSelf := &model.SelfUser{
		User: model.User{
			UUID:     testUUID,
			Username: newUsername,
			UserType: "manual",
		},
	}

	mockRepo.On("UpdateSelfUser", ctx, testUUID, oldPass, newUsername, newPass).
		Return(updatedSelf, nil)

	result, err := svc.UpdateSelfProfile(ctx, testUUID, oldPass, newUsername, newPass)

	assert.NoError(t, err)
	assert.Equal(t, newUsername, result.Username)
	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdateSelfProfile_Success_ChangePassword(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()
	oldPass := "oldpass"
	newUsername := ""
	newPass := "NewPass123"

	updatedSelf := &model.SelfUser{
		User: model.User{
			UUID:     testUUID,
			Username: "user",
			UserType: "manual",
		},
	}

	mockRepo.On("UpdateSelfUser", ctx, testUUID, oldPass, newUsername, newPass).
		Return(updatedSelf, nil)

	result, err := svc.UpdateSelfProfile(ctx, testUUID, oldPass, newUsername, newPass)

	assert.NoError(t, err)
	assert.Equal(t, updatedSelf.UUID, result.UUID)
	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdateSelfProfile_NewPasswordTooShort(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()

	_, err := svc.UpdateSelfProfile(ctx, testUUID, "old", "new", "123")

	assert.ErrorIs(t, err, ErrPasswordTooShort)
	mockRepo.AssertNotCalled(t, "UpdateSelfUser")
}

func TestUserService_UpdateSelfProfile_WrongOldPassword(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()

	mockRepo.On("UpdateSelfUser", ctx, testUUID, "wrong", "new", "").
		Return(nil, repository.ErrInvalidCredentials)

	_, err := svc.UpdateSelfProfile(ctx, testUUID, "wrong", "new", "")

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdateSelfProfile_SSO_CannotChangePassword(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()
	newPassword := "NewPass123"

	mockRepo.On("UpdateSelfUser", ctx, testUUID, "", "", newPassword).
		Return(nil, repository.ErrSSOUserPasswordChange)

	_, err := svc.UpdateSelfProfile(ctx, testUUID, "", "", newPassword)

	assert.ErrorIs(t, err, ErrSSOUserPasswordChange)
	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdateSelfProfile_UsernameConflict(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()

	mockRepo.On("UpdateSelfUser", ctx, testUUID, "pass", "taken", "").
		Return(nil, repository.ErrUserAlreadyExists)

	_, err := svc.UpdateSelfProfile(ctx, testUUID, "pass", "taken", "")

	assert.ErrorIs(t, err, ErrUserAlreadyExists)
	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdateSelfProfile_UserNotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()

	mockRepo.On("UpdateSelfUser", ctx, testUUID, "pass", "new", "").
		Return(nil, repository.ErrUserNotFound)

	_, err := svc.UpdateSelfProfile(ctx, testUUID, "pass", "new", "")

	assert.ErrorIs(t, err, repository.ErrUserNotFound)
	mockRepo.AssertExpectations(t)
}

func TestUserService_Logout_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()

	mockRepo.On("Logout", ctx, testUUID).Return(nil)

	err := svc.Logout(ctx, testUUID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
