package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Тест хеширования пароля
func TestHashPassword(t *testing.T) {
	//Тестовые данные
	password := "securepassword123"

	hash, err := HashPassword(password)

	//require.NoError — критическая проверка
	require.NoError(t, err, "HashPassword() should not return error")

	//assert.NotNil — проверяем что хеш не nil
	assert.NotNil(t, hash, "Hash should not be nil")

	//assert.NotEqual — хеш НЕ должен совпадать с паролем
	assert.NotEqual(t, password, string(hash), "Hash should not equal password")

	// Проверяем что это действительно bcrypt хеш
	assert.Contains(t, string(hash), "$2", "Should be bcrypt hash")
}

// Тест сравнения паролей
func TestComparePassword(t *testing.T) {
	//Тестовые данные
	password := "securepassword123"

	hash, err := HashPassword(password)
	require.NoError(t, err)

	// правильный пароль
	t.Run("correct password", func(t *testing.T) {
		err := ComparePassword(hash, password)

		//require.NoError — должно совпасть
		require.NoError(t, err, "Correct password should match")
	})

	//  неправильный пароль
	t.Run("wrong password", func(t *testing.T) {
		// Строка 62: Вызываем с неправильным паролем
		err := ComparePassword(hash, "wrongpassword")

		// assert.Error — ДОЛЖНА быть ошибка
		assert.Error(t, err, "Wrong password should not match")
	})

	// пустой пароль
	t.Run("empty password", func(t *testing.T) {
		err := ComparePassword(hash, "")
		assert.Error(t, err, "Empty password should not match")
	})
}

// Тест что каждый хеш уникален
func TestHashPassword_Unique(t *testing.T) {
	password := "samepassword"

	//Создаём два хеша одного пароля
	hash1, err1 := HashPassword(password)
	hash2, err2 := HashPassword(password)

	// Проверяем что обе успешны
	require.NoError(t, err1)
	require.NoError(t, err2)

	//Хеши должны быть РАЗНЫЕ
	assert.NotEqual(t, hash1, hash2, "Each hash should be unique (salt)")
	assert.NoError(t, ComparePassword(hash1, password))
	assert.NoError(t, ComparePassword(hash2, password))
}
