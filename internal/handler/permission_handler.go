package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	per "github.com/vega-trello/trello-back/internal/dto/permission" // Импортируем общий пакет DTO
)

type PermissionHandler struct {
	permissionService PermissionServiceInterface
}

func NewPermissionHandler(permissionService PermissionServiceInterface) *PermissionHandler {
	return &PermissionHandler{
		permissionService: permissionService,
	}
}

// ListPermissions GET /projects/permissions
// Возвращает список всех доступных системных прав
func (h *PermissionHandler) ListPermissions(c *gin.Context) {
	permissions, err := h.permissionService.GetAllPermissions(c.Request.Context())
	if err != nil {
		handleServiceError(c, err)
		return
	}

	responses := per.FromPermissionModels(permissions)

	c.JSON(http.StatusOK, responses)
}
