package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Тест валидации RegisterRequest
func TestRegisterRequest_Validate(t *testing.T) {
	//Таблица тестовых случаев
	tests := []struct {
		name        string
		req         RegisterRequest // Входные данные
		wantErr     bool
		errContains string
	}{
		// test валидные данные
		{
			name: "valid request",
			req: RegisterRequest{
				Username: "alex_dev",
				Password: "securepass123",
			},
			wantErr: false, // Ошибки не ожидаем
		},
		// test пустой username
		{
			name: "empty username",
			req: RegisterRequest{
				Username: "", // Пустой
				Password: "securepass123",
			},
			wantErr:     true,       // Ожидаем ошибку
			errContains: "username", // В сообщении должно быть "username"
		},
		// test короткий пароль
		{
			name: "short password",
			req: RegisterRequest{
				Username: "alex_dev",
				Password: "short", // Меньше 8 символов
			},
			wantErr:     true,
			errContains: "password",
		},
		// test невалидный username
		{
			name: "invalid username format",
			req: RegisterRequest{
				Username: "Alex@Dev", // Заглавные + спецсимвол
				Password: "securepass123",
			},
			wantErr:     true,
			errContains: "username",
		},
	}

	for _, tt := range tests {
		//  Подтест
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()

			// Проверяем наличие ошибки
			if tt.wantErr {
				// assert.Error — должна быть ошибка
				assert.Error(t, err, "Validate() should return error")

				// assert.Contains — проверяем текст ошибки
				// errContains задан — проверяем что он в сообщении
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				//assert.NoError — ошибки быть не должно
				assert.NoError(t, err, "Validate() should not return error")
			}
		})
	}
}

// Тест валидации LoginRequest
func TestLoginRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     LoginRequest
		wantErr bool
	}{
		{"valid", LoginRequest{Username: "alex", Password: "12345678"}, false},
		{"empty username", LoginRequest{Username: "", Password: "12345678"}, true},
		{"empty password", LoginRequest{Username: "alex", Password: ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
