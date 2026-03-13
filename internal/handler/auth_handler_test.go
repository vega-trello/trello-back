package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/vega-trello/trello-back/internal/dto/auth"
)

// MockAuthService — мок для сервиса
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, username, password string) (*dto.AuthResponse, error) {
	args := m.Called(ctx, username, password)

	var response *dto.AuthResponse
	if args.Get(0) != nil {
		response = args.Get(0).(*dto.AuthResponse)
	}
	return response, args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, username, password string) (*dto.AuthResponse, error) {
	args := m.Called(ctx, username, password)

	var response *dto.AuthResponse
	if args.Get(0) != nil {
		response = args.Get(0).(*dto.AuthResponse)
	}
	return response, args.Error(1)
}

// setupRouter — вспомогательная функция для создания тестового роутера
func setupRouter(mockService *MockAuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	handler := NewAuthHandler(mockService)

	router.POST("/api/v1/auth/register", handler.Register)
	router.POST("/api/v1/auth/login", handler.Login)

	return router
}

// Тест успешной регистрации
func TestAuthHandler_Register_Success(t *testing.T) {
	mockService := new(MockAuthService)

	router := setupRouter(mockService)

	// Тестовые данные
	reqBody := dto.RegisterRequest{
		Username: "testuser",
		Password: "securepassword123",
	}

	jsonData, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonData))

	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	//Ожидаемый ответ от сервиса
	expectedResponse := &dto.AuthResponse{
		Token: "dev-test-uuid",
		User: dto.UserInfo{
			UUID:      "550e8400-e29b-41d4-a716-446655440000",
			Username:  "testuser",
			UserType:  "manual",
			CreatedAt: getTestTime(),
		},
	}

	mockService.On("Register", mock.Anything, "testuser", "securepassword123").
		Return(expectedResponse, nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "Should return 201 Created")

	var response dto.AuthResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	//Проверяем данные
	assert.Equal(t, expectedResponse.Token, response.Token)
	assert.Equal(t, expectedResponse.User.Username, response.User.Username)

	//Проверяем что мок вызван
	mockService.AssertExpectations(t)
}

// Тест регистрации с невалидными данными
func TestAuthHandler_Register_InvalidInput(t *testing.T) {
	mockService := new(MockAuthService)
	router := setupRouter(mockService)

	// Невалидный запрос (короткий пароль)
	reqBody := dto.RegisterRequest{
		Username: "testuser",
		Password: "short", // Меньше 8 символов
	}

	jsonData, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Строка 166: Проверяем 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 for invalid input")

	//Проверяем что сервис НЕ вызвали
	// mockService.AssertNotCalled — проверяем что метод не вызывали
	mockService.AssertNotCalled(t, "Register")
}

// Тест логина успех
func TestAuthHandler_Login_Success(t *testing.T) {
	mockService := new(MockAuthService)
	router := setupRouter(mockService)

	reqBody := dto.LoginRequest{
		Username: "testuser",
		Password: "securepassword123",
	}

	jsonData, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	expectedResponse := &dto.AuthResponse{
		Token: "dev-test-uuid",
		User: dto.UserInfo{
			UUID:      "550e8400-e29b-41d4-a716-446655440000",
			Username:  "testuser",
			UserType:  "manual",
			CreatedAt: getTestTime(),
		},
	}

	mockService.On("Login", mock.Anything, "testuser", "securepassword123").
		Return(expectedResponse, nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK")

	var response dto.AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, expectedResponse.Token, response.Token)

	mockService.AssertExpectations(t)
}

// Тест логина с ошибкой
func TestAuthHandler_Login_Unauthorized(t *testing.T) {
	mockService := new(MockAuthService)
	router := setupRouter(mockService)

	reqBody := dto.LoginRequest{
		Username: "testuser",
		Password: "wrongpassword",
	}

	jsonData, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Сервис возвращает ошибку
	mockService.On("Login", mock.Anything, "testuser", "wrongpassword").
		Return(nil, errors.New("invalid username or password"))

	router.ServeHTTP(w, req)

	//Проверяем 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return 401 for wrong credentials")

	mockService.AssertExpectations(t)
}

func getTestTime() time.Time {
	return time.Date(2024, time.March, 1, 10, 0, 0, 0, time.UTC)
}
