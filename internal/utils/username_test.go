//go:build !integration
// +build !integration

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomUsername_Format(t *testing.T) {
	for i := 0; i < 10; i++ {
		username := GenerateRandomUsername()

		// Проверяем формат: {adjective}_{noun}_{number}
		assert.Regexp(t, `^[a-z]+_[a-z]+_\d{1,3}$`, username,
			"username should match format: {adjective}_{noun}_{number}")

		assert.LessOrEqual(t, len(username), 32, "username should not exceed 32 characters")
	}
}

func TestGenerateRandomUsername_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		username := GenerateRandomUsername()
		if seen[username] {
			t.Logf("Duplicate username generated: %s (iteration %d)", username, i)
		}
		seen[username] = true
	}
	assert.Greater(t, len(seen), 50, "should generate mostly unique usernames")
}
