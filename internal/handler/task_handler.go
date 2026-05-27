package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/task"
	"github.com/vega-trello/trello-back/internal/middleware"
)

type TaskHandler struct {
	taskService TaskServiceInterface
}

func NewTaskHandler(taskService TaskServiceInterface) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

// ListProjectTasks GET /projects/{projectUUID}/tasks
func (h *TaskHandler) ListProjectTasks(c *gin.Context) {
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

	var archived *bool
	if archivedStr := c.Query("archived"); archivedStr != "" {
		val := archivedStr == "true"
		archived = &val
	}

	tasks, err := h.taskService.GetProjectTasks(c.Request.Context(), projectUUID, userUUID, archived)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModels(tasks))
}

// CreateTask POST /projects/{projectUUID}/tasks
func (h *TaskHandler) CreateTask(c *gin.Context) {
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

	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	if req.ColumnID == nil || *req.ColumnID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_column_id", "column_id is required and must be positive")
		return
	}

	task, err := h.taskService.CreateTask(c.Request.Context(), projectUUID, userUUID, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.FromModel(task))
}

// GetTask GET /projects/{projectUUID}/task?taskID={id}
func (h *TaskHandler) GetTask(c *gin.Context) {
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

	task, err := h.taskService.GetTask(c.Request.Context(), projectUUID, taskID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(task))
}

// UpdateTask PATCH /projects/{projectUUID}/task?taskID={id}
func (h *TaskHandler) UpdateTask(c *gin.Context) {
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

	var req dto.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	updated, err := h.taskService.UpdateTask(c.Request.Context(), projectUUID, taskID, userUUID, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(updated))
}

// DeleteTask DELETE /projects/{projectUUID}/task?taskID={id}
func (h *TaskHandler) DeleteTask(c *gin.Context) {
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

	if err := h.taskService.DeleteTask(c.Request.Context(), projectUUID, taskID, userUUID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

/*
// MoveTask POST /projects/{projectUUID}/tasks/{taskID}/move
func (h *TaskHandler) MoveTask(c *gin.Context) {
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

	taskIDStr := c.Param("taskID")
	if taskIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_param", "taskID path parameter is required")
		return
	}

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil || taskID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_param", "taskID must be a positive integer")
		return
	}

	var req struct {
		TargetColumnID int `json:"target_column_id" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := h.taskService.MoveTask(c.Request.Context(), projectUUID, taskID, req.TargetColumnID, userUUID); err != nil {
		handleServiceError(c, err)
		return
	}

	updated, err := h.taskService.GetTask(c.Request.Context(), projectUUID, taskID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(updated))
}

// ArchiveTask POST /projects/{projectUUID}/tasks/{taskID}/archive
func (h *TaskHandler) ArchiveTask(c *gin.Context) {
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

	taskIDStr := c.Param("taskID")
	if taskIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_param", "taskID path parameter is required")
		return
	}

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil || taskID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_param", "taskID must be a positive integer")
		return
	}

	var req struct {
		Archive bool `json:"archive" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := h.taskService.ArchiveTask(c.Request.Context(), projectUUID, taskID, userUUID, req.Archive); err != nil {
		handleServiceError(c, err)
		return
	}

	updated, err := h.taskService.GetTask(c.Request.Context(), projectUUID, taskID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(updated))
}
*/
