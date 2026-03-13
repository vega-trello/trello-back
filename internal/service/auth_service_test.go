package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vega-trello/trello-back/internal/utils"

	"github.com/vega-trello/trello-back/internal/dto/auth"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateManualUser(ctx context.Context, username string, passwordHash []byte) (*dto.UserInfo, error) {
	args := m.Called(ctx, username, passwordHash)

	var userInfo *dto.UserInfo
	if args.Get(0) != nil {
		userInfo = args.Get(0).(*dto.UserInfo)
	}

	return userInfo, args.Error(1)
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username string) (*dto.UserInfo, []byte, error) {
	args := m.Called(ctx, username)

	var userInfo *dto.UserInfo
	if args.Get(0) != nil {
		userInfo = args.Get(0).(*dto.UserInfo)
	}

	var passwordHash []byte
	if args.Get(1) != nil {
		passwordHash = args.Get(1).([]byte)
	}

	return userInfo, passwordHash, args.Error(2)
}

// Тест регистрации
func TestAuthService_Register(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewAuthService(mockRepo)

	ctx := context.Background()
	username := "testuser"
	password := "securepassword123"

	expectedUserInfo := &dto.UserInfo{
		UUID:      "550e8400-e29b-41d4-a716-446655440000",
		Username:  username,
		UserType:  "manual",
		CreatedAt: getTestTime(),
	}

	mockRepo.On("CreateManualUser", ctx, username, mock.Anything).
		Return(expectedUserInfo, nil)

	response, err := service.Register(ctx, username, password)

	//require.NoError — критическая проверка
	require.NoError(t, err, "Register() should not return error")

	//require.NotNil — response не должен быть nil
	require.NotNil(t, response, "Response should not be nil")

	//assert.Equal — проверяем данные
	assert.Equal(t, expectedUserInfo.UUID, response.User.UUID)
	assert.Equal(t, expectedUserInfo.Username, response.User.Username)
	assert.Equal(t, "manual", response.User.UserType)

	//assert.NotEmpty — токен не должен быть пустым
	assert.NotEmpty(t, response.Token, "Token should not be empty")

	//assert.Contains — токен должен начинаться с "dev-"
	assert.Contains(t, response.Token, "dev-", "Token should start with dev-")

	mockRepo.AssertExpectations(t)
}

// Тест регистрации с ошибкой (username занят)
func TestAuthService_Register_Error(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewAuthService(mockRepo)

	ctx := context.Background()
	username := "existinguser"
	password := "securepassword123"

	// Настраиваем мок вернуть ошибку
	// errors.New("username taken") — ошибка которую вернёт мок
	mockRepo.On("CreateManualUser", ctx, username, mock.Anything).
		Return(nil, errors.New("username taken"))

	response, err := service.Register(ctx, username, password)

	//assert.Error — ДОЛЖНА быть ошибка
	assert.Error(t, err, "Register() should return error for taken username")

	//assert.Nil — response должен быть nil
	assert.Nil(t, response, "Response should be nil on error")

	//Проверяем что мок вызван
	mockRepo.AssertExpectations(t)
}

// Тест логина
func TestAuthService_Login_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewAuthService(mockRepo)

	ctx := context.Background()
	username := "testuser"
	password := "securepassword123"

	// Хешируем пароль для теста
	passwordHash, _ := HashPasswordForTest(password)

	expectedUserInfo := &dto.UserInfo{
		UUID:      "550e8400-e29b-41d4-a716-446655440000",
		Username:  username,
		UserType:  "manual",
		CreatedAt: getTestTime(),
	}

	// Настраиваем мок вернуть пользователя + хеш
	mockRepo.On("FindByUsername", ctx, username).
		Return(expectedUserInfo, passwordHash, nil)

	response, err := service.Login(ctx, username, password)

	// Проверяем успех
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, expectedUserInfo.UUID, response.User.UUID)

	mockRepo.AssertExpectations(t)
}

// Тест логина с неправильным паролем
func TestAuthService_Login_WrongPassword(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewAuthService(mockRepo)

	ctx := context.Background()
	username := "testuser"
	wrongPassword := "wrongpassword123"

	//Хеш от ДРУГОГО пароля
	correctPasswordHash, _ := HashPasswordForTest("correctpassword")

	// Возвращаем хеш который не совпадёт с wrongPassword
	mockRepo.On("FindByUsername", ctx, username).
		Return(&dto.UserInfo{Username: username}, correctPasswordHash, nil)

	// Вызываем Login
	response, err := service.Login(ctx, username, wrongPassword)

	// ДОЛЖНА быть ошибка
	assert.Error(t, err, "Login should fail with wrong password")
	assert.Nil(t, response)

	//Проверяем текст ошибки (общая ошибка для безопасности)
	assert.Contains(t, err.Error(), "invalid username or password")

	mockRepo.AssertExpectations(t)
}

// Тест логина с несуществующим пользователем
func TestAuthService_Login_UserNotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewAuthService(mockRepo)

	ctx := context.Background()
	username := "nonexistent"

	//Репозиторий возвращает ошибку (пользователь не найден)
	mockRepo.On("FindByUsername", ctx, username).
		Return(nil, nil, errors.New("user not found"))

	response, err := service.Login(ctx, username, "anypassword")

	// ДОЛЖНА быть ошибка
	assert.Error(t, err)
	assert.Nil(t, response)

	//Общая ошибка (не говорим "user not found")
	assert.Contains(t, err.Error(), "invalid username or password")

	mockRepo.AssertExpectations(t)
}

// getTestTime — возвращает фиксированное время для тестов
func getTestTime() time.Time {
	// time.Date — создаём конкретную дату (не time.Now() чтобы было предсказуемо)
	// 2024, March, 1, 10, 0, 0, 0, time.UTC — фиксированное время
	return time.Date(2024, time.March, 1, 10, 0, 0, 0, time.UTC)
}

// Вспомогательная функция для хеширования в тестах
func HashPasswordForTest(password string) ([]byte, error) {
	return utils.HashPassword(password)
}
