package utils

import (
	"regexp"
	"strings"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// Регулярка: первая буква [a-z], потом 2-63 символа [a-z0-9_-]
var UsernameRegex = regexp.MustCompile(`^[a-z][a-z0-9_-]{2,63}$`)

func ValidateUsername(username string) error {
	username = strings.ToLower(username)

	if !UsernameRegex.MatchString(username) {
		return &ValidationError{
			Field:   "username",
			Message: "can contain only lowercase letters, numbers, underscore and dash, 3-64 chars",
		}
	}
	return nil
}

func IsValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 64 {
		return false
	}
	return UsernameRegex.MatchString(username)
}

func ValidatePassword(password string) error {
	if password == "" {
		return &ValidationError{
			Field:   "password",
			Message: "is required",
		}
	}
	if len(password) < 8 {
		return &ValidationError{
			Field:   "password",
			Message: "must be at least 8 characters",
		}
	}
	return nil
}

func NormalizeUsername(username string) string {
	username = strings.TrimSpace(username)
	username = strings.ToLower(username)
	return username
}
