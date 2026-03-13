package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		//Тестовый случай
		// "valid username" — название (покажется в отчёте если тест упадёт)
		// "alex_dev" — валидный username (буквы, цифры, подчёркивание)
		// true — ожидаем что IsValidUsername вернёт true
		{"valid username", "alex_dev", true},

		// Тестовый случай
		// "valid with numbers" — username с цифрами
		// "user123" — тоже валидный
		{"valid with numbers", "user123", true},

		// Тестовый случай
		// "valid with dash" — username с дефисом
		{"valid with dash", "user-name", true},

		// Тестовый случай
		// "invalid uppercase" — НЕвалидный (заглавная буква)
		// "Alex" — содержит заглавную A  должно вернуть false
		{"invalid uppercase", "Alex", false},

		// Тестовый случай
		// "invalid special chars" — спецсимволы запрещены
		// "alex@dev" — @ запрещён false
		{"invalid special chars", "alex@dev", false},

		//Тестовый случай
		// "invalid spaces" — пробелы запрещены
		{"invalid spaces", "alex dev", false},

		//Тестовый случай
		// "too short" — меньше 3 символов
		// "al" — 2 символа → false
		{"too short", "al", false},

		// Тестовый случай
		// "empty" — пустая строка
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidUsername(tt.username)
			assert.Equal(t, tt.want, got, "IsValidUsername()")
		})
	}
}

// Тест для NormalizeUsername
func TestNormalizeUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// тест заглавные буквы
		{"converts to lowercase", "Alex", "alex"},

		//тест пробелы
		{"trims spaces", "  alex  ", "alex"},

		// уже нормальный тест
		{"already normalized", "alex", "alex"},

		//пустая строка
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeUsername(tt.input)
			assert.Equal(t, tt.expected, got, "NormalizeUsername()")
		})
	}
}
