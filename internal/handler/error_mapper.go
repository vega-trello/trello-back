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
	//User errors
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

	// Project errors
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

	// 🔹 Column errors
	case errors.Is(err, service.ErrColumnNotFound):
		respondError(c, http.StatusNotFound, "column_not_found", err.Error())
	case errors.Is(err, service.ErrColumnHasTasks):
		respondError(c, http.StatusConflict, "column_has_tasks", err.Error())
	case errors.Is(err, service.ErrInvalidColumnName):
		respondError(c, http.StatusBadRequest, "invalid_column_name", err.Error())
	case errors.Is(err, service.ErrInvalidPosition):
		respondError(c, http.StatusBadRequest, "invalid_position", err.Error())
	case errors.Is(err, service.ErrInvalidDirection):
		respondError(c, http.StatusBadRequest, "invalid_direction", err.Error())

	//Task errors
	case errors.Is(err, service.ErrTaskNotFound):
		respondError(c, http.StatusNotFound, "task_not_found", err.Error())
	case errors.Is(err, service.ErrInvalidTitle):
		respondError(c, http.StatusBadRequest, "invalid_task_title", err.Error())
	case errors.Is(err, service.ErrInvalidDescriptionTask):
		respondError(c, http.StatusBadRequest, "invalid_task_description", err.Error())
	case errors.Is(err, service.ErrInvalidDateRange):
		respondError(c, http.StatusBadRequest, "invalid_date_range", err.Error())
	case errors.Is(err, service.ErrInvalidDateFormat):
		respondError(c, http.StatusBadRequest, "invalid_date_format", err.Error())
	case errors.Is(err, service.ErrInvalidColumn):
		respondError(c, http.StatusBadRequest, "invalid_column", err.Error())
	case errors.Is(err, service.ErrInvalidStatus):
		respondError(c, http.StatusBadRequest, "invalid_status", err.Error())

	//Member errors
	case errors.Is(err, service.ErrMemberNotFound):
		respondError(c, http.StatusNotFound, "member_not_found", err.Error())
	case errors.Is(err, service.ErrMemberAlreadyExists):
		respondError(c, http.StatusConflict, "member_already_exists", err.Error())
	case errors.Is(err, service.ErrCannotRemoveLastOwner):
		respondError(c, http.StatusConflict, "cannot_remove_last_owner", err.Error())
	case errors.Is(err, service.ErrCannotRemoveSelf):
		respondError(c, http.StatusForbidden, "cannot_remove_self", err.Error())
	case errors.Is(err, service.ErrInvalidRole):
		respondError(c, http.StatusBadRequest, "invalid_role", err.Error())
	case errors.Is(err, service.ErrInvalidUUID):
		respondError(c, http.StatusBadRequest, "invalid_uuid_format", err.Error())

	// internal server error
	default:
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error()) // потом сделать чтобы возвращал общую ошибку
	}
}

func respondError(c *gin.Context, status int, code string, msg string) {
	c.AbortWithStatusJSON(status, dto.ErrorResponse{
		Error:   code,
		Message: msg,
	})
}
