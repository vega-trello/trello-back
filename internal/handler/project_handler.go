package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/project"
	"github.com/vega-trello/trello-back/internal/middleware"
)

type ProjectHandler struct {
	projectService ProjectServiceInterface
}

func NewProjectHandler(projectService ProjectServiceInterface) *ProjectHandler {
	return &ProjectHandler{projectService: projectService}
}

// ListProjects GET /projects
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	projects, err := h.projectService.GetUserProjects(c.Request.Context(), userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModels(projects))
}

// CreateProject POST /projects
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	var req dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	project, err := h.projectService.CreateProject(
		c.Request.Context(),
		userUUID,
		req.Title,
		req.Description,
	)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.FromModel(project))
}

// GetProject GET /projects/{projectUUID}
func (h *ProjectHandler) GetProject(c *gin.Context) {
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

	project, err := h.projectService.GetProject(c.Request.Context(), projectUUID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(project))
}

// UpdateProject PATCH /projects/{projectUUID}
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
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

	var req dto.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// 🔹 Дополнительная валидация
	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	updated, err := h.projectService.UpdateProject(
		c.Request.Context(),
		projectUUID,
		userUUID,
		req.Title,       // *string (nil = не менять)
		req.Description, // *string (nil = не менять)
	)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(updated))
}

// DeleteProject DELETE /projects/{projectUUID}
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
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

	if err := h.projectService.DeleteProject(c.Request.Context(), projectUUID, userUUID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
