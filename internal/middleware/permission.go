package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/user"
	"github.com/vega-trello/trello-back/internal/service"
)

func RequirePermission(checker service.PermissionChecker, requiredPerm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID, ok := GetUserUUID(c)
		if !ok {
			respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
			c.Abort()
			return
		}

		projectUUIDStr := c.Param("projectUUID")
		if projectUUIDStr == "" {
			c.Next()
			return
		}

		projectUUID, err := uuid.Parse(projectUUIDStr)
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid_uuid", "Invalid project UUID format")
			c.Abort()
			return
		}

		if err := checker.Check(c.Request.Context(), projectUUID, userUUID, requiredPerm); err != nil {
			if err == service.ErrPermissionDenied {
				respondError(c, http.StatusForbidden, "access_denied", "Insufficient permissions")
			} else {
				respondError(c, http.StatusInternalServerError, "internal_error", "Permission check failed")
			}
			c.Abort()
			return
		}

		c.Next()
	}
}

func respondError(c *gin.Context, status int, code string, msg string) {
	c.AbortWithStatusJSON(status, dto.ErrorResponse{
		Error:   code,
		Message: msg,
	})
}
