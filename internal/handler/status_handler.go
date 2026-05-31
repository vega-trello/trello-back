package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/status"
	"github.com/vega-trello/trello-back/internal/middleware"
)

type StatusHandler struct {
	statusService StatusServiceInterface
}

func NewStatusHandler(statusService StatusServiceInterface) *StatusHandler {
	return &StatusHandler{statusService: statusService}
}

// ListProjectStatuses GET /projects/{projectUUID}/statuses
func (h *StatusHandler) ListProjectStatuses(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	projectUUID, err := uuid.Parse(c.Param("projectUUID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_uuid", "Invalid project UUID format")
		return
	}

	statuses, err := h.statusService.GetProjectStatuses(c.Request.Context(), projectUUID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	statusesDTO := dto.FromModels(statuses)
	c.JSON(http.StatusOK, statusesDTO)
}

// CreateStatus POST /projects/{projectUUID}/statuses
func (h *StatusHandler) CreateStatus(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	projectUUID, err := uuid.Parse(c.Param("projectUUID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_uuid", "Invalid project UUID format")
		return
	}

	var req dto.CreateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	status, err := h.statusService.CreateStatus(c.Request.Context(), projectUUID, userUUID, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.FromModel(status))
}

// GetStatus GET /projects/{projectUUID}/statuses/{statusID}
func (h *StatusHandler) GetStatus(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	projectUUID, err := uuid.Parse(c.Param("projectUUID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_uuid", "Invalid project UUID format")
		return
	}

	statusIDStr := c.Param("statusID")
	if statusIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_status_id", "statusID path parameter is required")
		return
	}

	statusID, err := strconv.Atoi(statusIDStr)
	if err != nil || statusID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_status_id", "Invalid status ID format")
		return
	}

	status, err := h.statusService.GetStatus(c.Request.Context(), projectUUID, statusID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(status))
}

// UpdateStatus PATCH /projects/{projectUUID}/statuses/{statusID}
func (h *StatusHandler) UpdateStatus(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	projectUUID, err := uuid.Parse(c.Param("projectUUID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_uuid", "Invalid project UUID format")
		return
	}

	statusIDStr := c.Param("statusID")
	if statusIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_status_id", "statusID path parameter is required")
		return
	}

	statusID, err := strconv.Atoi(statusIDStr)
	if err != nil || statusID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_status_id", "Invalid status ID format")
		return
	}

	var req dto.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	updated, err := h.statusService.UpdateStatus(c.Request.Context(), projectUUID, statusID, userUUID, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(updated))
}

// DeleteStatus DELETE /projects/{projectUUID}/statuses/{statusID}
func (h *StatusHandler) DeleteStatus(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	projectUUID, err := uuid.Parse(c.Param("projectUUID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_uuid", "Invalid project UUID format")
		return
	}

	statusIDStr := c.Param("statusID")
	if statusIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_status_id", "statusID path parameter is required")
		return
	}

	statusID, err := strconv.Atoi(statusIDStr)
	if err != nil || statusID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_status_id", "Invalid status ID format")
		return
	}

	if err := h.statusService.DeleteStatus(c.Request.Context(), projectUUID, statusID, userUUID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
