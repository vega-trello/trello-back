package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/role"
	"github.com/vega-trello/trello-back/internal/middleware"
)

type RoleHandler struct {
	roleService RoleServiceInterface
}

func NewRoleHandler(roleService RoleServiceInterface) *RoleHandler {
	return &RoleHandler{roleService: roleService}
}

// ListProjectRoles GET /projects/{projectUUID}/roles
// Возвращает все роли проекта
func (h *RoleHandler) ListProjectRoles(c *gin.Context) {
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

	roles, err := h.roleService.GetProjectRoles(c.Request.Context(), projectUUID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModels(roles))
}

// CreateRole POST /projects/{projectUUID}/roles
// Создаёт новую роль в проекте
func (h *RoleHandler) CreateRole(c *gin.Context) {
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

	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	role, err := h.roleService.CreateRole(c.Request.Context(), projectUUID, userUUID, req.Name, &req.Description, req.PermissionIDs)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.FromModel(role))
}

// GetRole GET /projects/{projectUUID}/roles/{roleID}
// Возвращает роль по ID
func (h *RoleHandler) GetRole(c *gin.Context) {
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

	roleIDStr := c.Param("roleID")
	if roleIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_role_id", "roleID path parameter is required")
		return
	}

	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil || roleID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_role_id", "Invalid role ID format")
		return
	}

	role, err := h.roleService.GetRole(c.Request.Context(), projectUUID, roleID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(role))
}

// UpdateRole PATCH /projects/{projectUUID}/roles/{roleID}
// Обновляет роль с полной перезаписью разрешений
func (h *RoleHandler) UpdateRole(c *gin.Context) {
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

	roleIDStr := c.Param("roleID")
	if roleIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_role_id", "roleID path parameter is required")
		return
	}

	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil || roleID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_role_id", "Invalid role ID format")
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	updated, err := h.roleService.UpdateRole(c.Request.Context(), projectUUID, roleID, userUUID, req.Name, &req.Description, req.PermissionIDs)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.FromModel(updated))
}

// DeleteRole DELETE /projects/{projectUUID}/roles/{roleID}
// Удаляет роль с проверкой, что она не используется
func (h *RoleHandler) DeleteRole(c *gin.Context) {
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

	roleIDStr := c.Param("roleID")
	if roleIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_role_id", "roleID path parameter is required")
		return
	}

	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil || roleID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_role_id", "Invalid role ID format")
		return
	}

	if err := h.roleService.DeleteRole(c.Request.Context(), projectUUID, roleID, userUUID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetRolePermissions GET /projects/{projectUUID}/roles/{roleID}/permissions
// Возвращает список разрешений роли
func (h *RoleHandler) GetRolePermissions(c *gin.Context) {
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

	roleIDStr := c.Param("roleID")
	if roleIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_role_id", "roleID path parameter is required")
		return
	}

	roleID, err := strconv.Atoi(roleIDStr)
	if err != nil || roleID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_role_id", "Invalid role ID format")
		return
	}

	permissions, err := h.roleService.GetRolePermissions(c.Request.Context(), projectUUID, roleID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	permResponses := make([]dto.PermissionResponse, len(permissions))
	for i, p := range permissions {
		permResponses[i] = dto.FromPermissionModel(p)
	}

	c.JSON(http.StatusOK, permResponses)
}
