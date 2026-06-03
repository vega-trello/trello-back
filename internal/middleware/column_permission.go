package middleware

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vega-trello/trello-back/internal/service"
)

func RequireColumnPermission(
	checker service.PermissionChecker,
	db *pgxpool.Pool,
	requiredPerm string,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID, ok := GetUserUUID(c)
		if !ok {
			respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
			c.Abort()
			return
		}

		columnIDStr := c.Param("columnID")
		if columnIDStr == "" {
			respondError(c, http.StatusBadRequest, "invalid_column_id", "Column ID required")
			c.Abort()
			return
		}

		columnID, err := strconv.Atoi(columnIDStr)
		if err != nil || columnID <= 0 {
			respondError(c, http.StatusBadRequest, "invalid_column_id", "Invalid column ID format")
			c.Abort()
			return
		}

		var projectUUID uuid.UUID
		err = db.QueryRow(c.Request.Context(), `
			SELECT project_uuid FROM project_column WHERE id = $1
		`, columnID).Scan(&projectUUID)

		if errors.Is(err, pgx.ErrNoRows) {
			respondError(c, http.StatusNotFound, "column_not_found", "Column not found")
			c.Abort()
			return
		}
		if err != nil {
			respondError(c, http.StatusInternalServerError, "internal_error", "Failed to verify column permissions")
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
