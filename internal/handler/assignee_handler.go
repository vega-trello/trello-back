package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/assignee"
	"github.com/vega-trello/trello-back/internal/middleware"
)

type AssigneeHandler struct {
	assigneeService AssigneeServiceInterface
}

func NewAssigneeHandler(assigneeService AssigneeServiceInterface) *AssigneeHandler {
	return &AssigneeHandler{assigneeService: assigneeService}
}

// ListTaskAssignees GET /projects/{projectUUID}/assignees?taskID={id}
// Возвращает всех исполнителей задачи
func (h *AssigneeHandler) ListTaskAssignees(c *gin.Context) {
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
		respondError(c, http.StatusBadRequest, "missing_task_id", "taskID query parameter is required")
		return
	}

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil || taskID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_task_id", "Invalid task ID format")
		return
	}

	assignees, err := h.assigneeService.GetTaskAssignees(c.Request.Context(), projectUUID, taskID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	assigneesList := make([]dto.AssigneeResponse, 0, len(assignees))
	for _, a := range assignees {
		if a != nil {
			assigneesList = append(assigneesList, *a)
		}
	}

	c.JSON(http.StatusOK, dto.AssingeeResponse{
		Assignees: assigneesList,
		Total:     len(assigneesList),
	})
}

// AddAssignee POST /projects/{projectUUID}/assignees?taskID={id}
// Назначает пользователя исполнителем задачи
func (h *AssigneeHandler) AddAssignee(c *gin.Context) {
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
		respondError(c, http.StatusBadRequest, "missing_task_id", "taskID query parameter is required")
		return
	}

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil || taskID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_task_id", "Invalid task ID format")
		return
	}

	var req dto.CreateAssigneeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	if err := h.assigneeService.AssignUserToTask(c.Request.Context(), projectUUID, taskID, userUUID, req); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusCreated)
}

// RemoveAssignee DELETE /projects/{projectUUID}/assignee?taskID={id}&userUUID={uuid}
// Удаляет исполнителя из задачи
func (h *AssigneeHandler) RemoveAssignee(c *gin.Context) {
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
		respondError(c, http.StatusBadRequest, "missing_task_id", "taskID query parameter is required")
		return
	}

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil || taskID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_task_id", "Invalid task ID format")
		return
	}

	assigneeUUIDStr := c.Query("userUUID")
	if assigneeUUIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_user_uuid", "userUUID query parameter is required")
		return
	}

	assigneeUUID, err := uuid.Parse(assigneeUUIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_user_uuid", "Invalid user UUID format")
		return
	}

	if err := h.assigneeService.RemoveAssignee(c.Request.Context(), projectUUID, taskID, userUUID, assigneeUUID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
