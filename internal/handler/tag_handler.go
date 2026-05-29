package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/tag"
	"github.com/vega-trello/trello-back/internal/middleware"
)

// TagHandler обрабатывает HTTP-запросы, связанные с тегами
type TagHandler struct {
	tagService TagServiceInterface
}

// NewTagHandler создаёт новый экземпляр TagHandler
func NewTagHandler(tagService TagServiceInterface) *TagHandler {
	return &TagHandler{tagService: tagService}
}

// ListProjectTags GET /projects/{projectUUID}/tag
// Возвращает все теги проекта
func (h *TagHandler) ListProjectTags(c *gin.Context) {
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

	tags, err := h.tagService.GetProjectTags(c.Request.Context(), projectUUID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModels(tags))
}

// CreateTag POST /projects/{projectUUID}/tag
// Создаёт новый тег в проекте
func (h *TagHandler) CreateTag(c *gin.Context) {
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

	var req dto.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	tag, err := h.tagService.CreateTag(c.Request.Context(), projectUUID, userUUID, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.FromModel(tag))
}

// UpdateTag PATCH /projects/{projectUUID}/tag?tagID={id}
// Обновляет имя и/или цвет тега
func (h *TagHandler) UpdateTag(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	_, err := uuid.Parse(c.Param("projectUUID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_uuid", "Invalid project UUID format")
		return
	}

	tagIDStr := c.Query("tagID")
	if tagIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_param", "tagID query parameter is required")
		return
	}

	tagID, err := strconv.Atoi(tagIDStr)
	if err != nil || tagID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_param", "tagID must be a positive integer")
		return
	}

	var req dto.UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	updated, err := h.tagService.UpdateTag(c.Request.Context(), tagID, userUUID, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(updated))
}

// DeleteTag DELETE /projects/{projectUUID}/tag?tagID={id}
// Удаляет тег из проекта
func (h *TagHandler) DeleteTag(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	_, err := uuid.Parse(c.Param("projectUUID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_uuid", "Invalid project UUID format")
		return
	}

	tagIDStr := c.Query("tagID")
	if tagIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_param", "tagID query parameter is required")
		return
	}

	tagID, err := strconv.Atoi(tagIDStr)
	if err != nil || tagID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_param", "tagID must be a positive integer")
		return
	}

	if err := h.tagService.DeleteTag(c.Request.Context(), tagID, userUUID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListTaskTags GET /projects/{projectUUID}/task/tags?taskID={id}
// Возвращает теги, привязанные к конкретной задаче
func (h *TagHandler) ListTaskTags(c *gin.Context) {
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

	taskIDStr := c.Query("taskID")
	if taskIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_param", "taskID query parameter is required")
		return
	}

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil || taskID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_param", "taskID must be a positive integer")
		return
	}

	tags, err := h.tagService.GetTaskTags(c.Request.Context(), projectUUID, taskID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModels(tags))
}

// AddTagToTask POST /projects/{projectUUID}/task/tags?taskID={id}&tagID={id}
// Привязывает тег к задаче
func (h *TagHandler) AddTagToTask(c *gin.Context) {
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

	taskIDStr := c.Query("taskID")
	if taskIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_param", "taskID query parameter is required")
		return
	}

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil || taskID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_param", "taskID must be a positive integer")
		return
	}

	tagIDStr := c.Query("tagID")
	if tagIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_param", "tagID query parameter is required")
		return
	}

	tagID, err := strconv.Atoi(tagIDStr)
	if err != nil || tagID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_param", "tagID must be a positive integer")
		return
	}

	if err := h.tagService.AddTagToTask(c.Request.Context(), projectUUID, userUUID, taskID, tagID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// RemoveTagFromTask DELETE /projects/{projectUUID}/task/tags?taskID={id}&tagID={id}
// Отвязывает тег от задачи
func (h *TagHandler) RemoveTagFromTask(c *gin.Context) {
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

	tagIDStr := c.Query("tagID")
	if tagIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_param", "tagID query parameter is required")
		return
	}

	tagID, err := strconv.Atoi(tagIDStr)
	if err != nil || tagID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_param", "tagID must be a positive integer")
		return
	}

	taskIDStr := c.Query("taskID")
	if taskIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_param", "taskID query parameter is required")
		return
	}

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil || taskID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_param", "taskID must be a positive integer")
		return
	}

	if err := h.tagService.RemoveTagFromTask(c.Request.Context(), projectUUID, userUUID, taskID, tagID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
