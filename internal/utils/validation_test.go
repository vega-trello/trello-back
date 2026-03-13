// Строка 1: Объявляем пакет utils
// В Go тесты находятся в том же пакете что и тестируемый код
package utils

// Строки 4-7: Импортируем пакеты для тестирования
import (
	// testing — стандартный пакет Go для тестов
	"testing"

	// github.com/stretchr/testify/assert — библиотека для красивых утверждений
	// Упрощает написание тестов (assert.Equal вместо if x != y { t.Error() })
	"github.com/stretchr/testify/assert"
)

// Строка 11: Функция теста
// func Test... — любая функция начинающаяся с Test считается тестом
// TestIsValidUsername — имя теста (должно описывать что тестируем)
// (t *testing.T) — параметр t (test), через него сообщаем об ошибках
func TestIsValidUsername(t *testing.T) {
	// Строка 17: Создаём таблицу тестовых случаев
	// tests — слайс (массив) структур с тестовыми данными
	// []struct{...} — анонимная структура для тестовых случаев
	// name — название подтеста (для отчёта)
	// username — входные данные
	// want — ожидаемый результат (true/false)
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		// Строка 26: Тестовый случай 1
		// "valid username" — название (покажется в отчёте если тест упадёт)
		// "alex_dev" — валидный username (буквы, цифры, подчёркивание)
		// true — ожидаем что IsValidUsername вернёт true
		{"valid username", "alex_dev", true},

		// Строка 27: Тестовый случай 2
		// "valid with numbers" — username с цифрами
		// "user123" — тоже валидный
		{"valid with numbers", "user123", true},

		// Строка 28: Тестовый случай 3
		// "valid with dash" — username с дефисом
		{"valid with dash", "user-name", true},

		// Строка 29: Тестовый случай 4
		// "invalid uppercase" — НЕвалидный (заглавная буква)
		// "Alex" — содержит заглавную A → должно вернуть false
		{"invalid uppercase", "Alex", false},

		// Строка 30: Тестовый случай 5
		// "invalid special chars" — спецсимволы запрещены
		// "alex@dev" — @ запрещён → false
		{"invalid special chars", "alex@dev", false},

		// Строка 31: Тестовый случай 6
		// "invalid spaces" — пробелы запрещены
		{"invalid spaces", "alex dev", false},

		// Строка 32: Тестовый случай 7
		// "too short" — меньше 3 символов
		// "al" — 2 символа → false
		{"too short", "al", false},

		// Строка 33: Тестовый случай 8
		// "empty" — пустая строка
		// "" — пустая → false
		{"empty", "", false},
	}
	// Закрывающая скобка слайса (строка 35)

	// Строка 38: Цикл по всем тестовым случаям
	// for _, tt := range tests — проходим по каждому тесту
	// _ — игнорируем индекс (не нужен)
	// tt — текущий тестовый случай (table test)
	for _, tt := range tests {
		// Строка 41: t.Run — создаёт подтест
		// tt.name — название подтеста (например "valid username")
		// func(t *testing.T) { ... } — функция теста
		// Каждый подтест запускается отдельно (можно запускать по имени)
		t.Run(tt.name, func(t *testing.T) {
			// Строка 45: Вызываем тестируемую функцию
			// got — переменная для полученного результата (got = получил)
			// IsValidUsername(tt.username) — вызов функции с тестовыми данными
			got := IsValidUsername(tt.username)

			// Строка 48: Утверждение (assertion)
			// assert.Equal — проверяет что два значения равны
			// t — *testing.T (куда писать ошибки)
			// tt.want — ожидаемое значение (want = хотим)
			// got — полученное значение
			// "IsValidUsername()" — сообщение которое покажется если тест упадёт
			assert.Equal(t, tt.want, got, "IsValidUsername()")
		})
		// Закрывающая скобка t.Run (строка 51)
	}
	// Закрывающая скобка for (строка 52)
}

// Закрывающая скобка функции (строка 53)

// Строка 56: Тест для NormalizeUsername
func TestNormalizeUsername(t *testing.T) {
	// Строка 58: Таблица тестовых случаев
	tests := []struct {
		name     string
		input    string // Входные данные
		expected string // Ожидаемый результат
	}{
		// Строка 64: Тест 1 — заглавные буквы
		// "Alex" → "alex" (lowercase)
		{"converts to lowercase", "Alex", "alex"},

		// Строка 65: Тест 2 — пробелы
		// "  alex  " → "alex" (trim + lowercase)
		{"trims spaces", "  alex  ", "alex"},

		// Строка 66: Тест 3 — уже нормальный
		// "alex" → "alex" (без изменений)
		{"already normalized", "alex", "alex"},

		// Строка 67: Тест 4 — пустая строка
		// "" → "" (остаётся пустой)
		{"empty string", "", ""},
	}
	// Закрывающая скопка слайса (строка 69)

	// Строка 72: Цикл по тестам
	for _, tt := range tests {
		// Строка 73: Подтест
		t.Run(tt.name, func(t *testing.T) {
			// Строка 75: Вызываем функцию
			got := NormalizeUsername(tt.input)

			// Строка 77: Проверяем результат
			assert.Equal(t, tt.expected, got, "NormalizeUsername()")
		})
	}
}

// Закрывающая скобка функции (строка 82)
