package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	dto "github.com/vega-trello/trello-back/internal/dto/user"
	"github.com/vega-trello/trello-back/internal/repository"
	"github.com/vega-trello/trello-back/internal/service"
)

// handleServiceError маппит доменные ошибки сервиса на HTTP-коды и возвращает клиенту
// Используется во всех хендлерах для единообразной обработки ошибок
func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		respondError(c, http.StatusUnauthorized, "invalid_credentials", err.Error())

	case errors.Is(err, service.ErrPasswordTooShort):
		respondError(c, http.StatusBadRequest, "password_too_short", err.Error())

	case errors.Is(err, service.ErrUserAlreadyExists):
		respondError(c, http.StatusConflict, "username_taken", err.Error())

	case errors.Is(err, service.ErrSSOUserPasswordChange):
		respondError(c, http.StatusForbidden, "sso_password_change_forbidden", err.Error())

	case errors.Is(err, repository.ErrUserNotFound):
		respondError(c, http.StatusNotFound, "user_not_found", "User not found")

	default:
		// Логируем ошибку на сервере (в будущем — через zap/slog)
		// c.Error(err) // для Gin middleware
		respondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}

// respondError формирует унифицированный ответ об ошибке в формате DTO
// и прерывает выполнение цепочки хендлеров
func respondError(c *gin.Context, status int, code string, msg string) {
	c.AbortWithStatusJSON(status, dto.ErrorResponse{
		Error:   code,
		Message: msg,
	})
}
