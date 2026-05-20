package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/column"
	"github.com/vega-trello/trello-back/internal/middleware"
)

type ColumnHandler struct {
	columnService ColumnServiceInterface
}

func NewColumnHandler(columnService ColumnServiceInterface) *ColumnHandler {
	return &ColumnHandler{columnService: columnService}
}

// ListProjectColumns GET /projects/{projectUUID}/columns
func (h *ColumnHandler) ListProjectColumns(c *gin.Context) {
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

	columns, err := h.columnService.GetProjectColumns(c.Request.Context(), projectUUID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModels(columns))
}

// CreateColumn POST /projects/{projectUUID}/columns
func (h *ColumnHandler) CreateColumn(c *gin.Context) {
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

	var req dto.CreateColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	column, err := h.columnService.CreateColumn(c.Request.Context(), projectUUID, userUUID, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.FromModel(column))
}

// GetColumn GET /columns/{columnID}
func (h *ColumnHandler) GetColumn(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	columnID, err := strconv.Atoi(c.Param("columnID"))
	if err != nil || columnID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_column_id", "Invalid column ID format")
		return
	}

	column, err := h.columnService.GetColumn(c.Request.Context(), columnID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(column))
}

// UpdateColumn PATCH /columns/{columnID}
// Поддерживает частичное обновление - передаются только изменённые поля
func (h *ColumnHandler) UpdateColumn(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	columnID, err := strconv.Atoi(c.Param("columnID"))
	if err != nil || columnID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_column_id", "Invalid column ID format")
		return
	}

	var req dto.UpdateColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	updated, err := h.columnService.UpdateColumn(c.Request.Context(), columnID, userUUID, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(updated))
}

// DeleteColumn DELETE /columns/{columnID}
func (h *ColumnHandler) DeleteColumn(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	columnID, err := strconv.Atoi(c.Param("columnID"))
	if err != nil || columnID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_column_id", "Invalid column ID format")
		return
	}

	if err := h.columnService.DeleteColumn(c.Request.Context(), columnID, userUUID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// MoveColumn POST /columns/{columnID}/move
func (h *ColumnHandler) MoveColumn(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	columnID, err := strconv.Atoi(c.Param("columnID"))
	if err != nil || columnID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_column_id", "Invalid column ID format")
		return
	}

	var req dto.MoveColumnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	updated, err := h.columnService.MoveColumn(c.Request.Context(), columnID, userUUID, req.Direction)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(updated))
}
