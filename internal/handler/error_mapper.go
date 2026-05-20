package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	dto "github.com/vega-trello/trello-back/internal/dto/user"
	"github.com/vega-trello/trello-back/internal/repository"
	"github.com/vega-trello/trello-back/internal/service"
)

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
	case errors.Is(err, service.ErrProjectNotFound):
		respondError(c, http.StatusNotFound, "project_not_found", err.Error())
	case errors.Is(err, service.ErrAccessDenied):
		respondError(c, http.StatusForbidden, "access_denied", err.Error())
	case errors.Is(err, service.ErrProjectTitleTaken):
		respondError(c, http.StatusConflict, "project_title_taken", err.Error())
	case errors.Is(err, service.ErrProjectHasMembers):
		respondError(c, http.StatusConflict, "project_has_members", err.Error())
	case errors.Is(err, service.ErrInvalidProjectTitle):
		respondError(c, http.StatusBadRequest, "invalid_project_title", err.Error())
	case errors.Is(err, service.ErrInvalidDescriptionProject):
		respondError(c, http.StatusBadRequest, "invalid_project_description", err.Error())

	default:
		respondError(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
	}
}

func respondError(c *gin.Context, status int, code string, msg string) {
	c.AbortWithStatusJSON(status, dto.ErrorResponse{
		Error:   code,
		Message: msg,
	})
}
