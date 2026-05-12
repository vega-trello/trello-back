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
	"github.com/vega-trello/trello-back/internal/model"
	"golang.org/x/crypto/bcrypt"
)

// Локальная ошибка для тестов, чтобы не нарушать DIP импортом пакета repository
var errNotFound = errors.New("user not found")

// MockUserRepository — мок для интерфейса service.UserRepository
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

func (m *MockUserRepository) FindOrCreateUserBySSO(ctx context.Context, provider, externalID, username string) (*model.User, error) {
	args := m.Called(ctx, provider, externalID, username)
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

func (m *MockUserRepository) UpdateSelfUser(ctx context.Context, userUUID uuid.UUID, oldPassword, newUsername, newPassword string) (*model.SelfUser, error) {
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

// ==================== TESTS ====================

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
	}

	mockRepo.On("RegisterPasswordUser", ctx, username, mock.AnythingOfType("[]uint8")).
		Return(expectedUser, nil)

	user, err := svc.Register(ctx, username, password)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser.UUID, user.UUID)
	assert.Equal(t, expectedUser.Username, user.Username)
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

func TestUserService_Login_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	username := "loginuser"
	password := "CorrectPass1"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	testUUID := uuid.New()
	expectedUser := &model.User{
		UUID:     testUUID,
		Username: username,
	}

	mockRepo.On("FindUserByUsername", ctx, username).
		Return(expectedUser, hash, nil)

	result, err := svc.Login(ctx, username, password)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, result)
	mockRepo.AssertExpectations(t)
}

func TestUserService_Login_WrongPassword(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	username := "user"
	hash, _ := bcrypt.GenerateFromPassword([]byte("RealPass1"), bcrypt.DefaultCost)

	mockRepo.On("FindUserByUsername", ctx, username).
		Return(&model.User{}, hash, nil)

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
		Return(nil, nil, errNotFound)

	_, err := svc.Login(ctx, username, "pass")

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetProfile_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()

	expectedSelf := &model.SelfUser{
		User: model.User{
			UUID:     testUUID,
			Username: "profile",
		},
		UserType: "manual",
	}

	mockRepo.On("GetSelfUser", ctx, testUUID).Return(expectedSelf, nil)

	result, err := svc.GetProfile(ctx, testUUID)

	assert.NoError(t, err)
	assert.Equal(t, expectedSelf, result)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetProfile_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()

	mockRepo.On("GetSelfUser", ctx, testUUID).Return(nil, errNotFound)

	_, err := svc.GetProfile(ctx, testUUID)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdateProfile_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()
	oldPass := "oldpass"
	newUsername := "newname"
	newPass := "NewPass123"

	updatedSelf := &model.SelfUser{
		User: model.User{
			UUID:     testUUID,
			Username: newUsername,
		},
		UserType: "manual",
	}

	mockRepo.On("UpdateSelfUser", ctx, testUUID, oldPass, newUsername, newPass).
		Return(updatedSelf, nil)

	result, err := svc.UpdateProfile(ctx, testUUID, oldPass, newUsername, newPass)

	assert.NoError(t, err)
	assert.Equal(t, updatedSelf, result)
	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdateProfile_NewPasswordTooShort(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()

	_, err := svc.UpdateProfile(ctx, testUUID, "old", "new", "123")

	assert.ErrorIs(t, err, ErrPasswordTooShort)
	mockRepo.AssertNotCalled(t, "UpdateSelfUser")
}

func TestUserService_UpdateProfile_RepoError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := NewUserService(mockRepo)

	ctx := context.Background()
	testUUID := uuid.New()
	repoErr := errors.New("constraint violation")

	mockRepo.On("UpdateSelfUser", ctx, testUUID, "old", "new", "StrongPass1").
		Return(nil, repoErr)

	_, err := svc.UpdateProfile(ctx, testUUID, "old", "new", "StrongPass1")

	assert.ErrorIs(t, err, repoErr)
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
