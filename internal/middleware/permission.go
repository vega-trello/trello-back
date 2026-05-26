package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	dto "github.com/vega-trello/trello-back/internal/dto/user"
)

// RequirePermission возвращает middleware, который проверяет наличие права у пользователя
func RequirePermission(requiredPerm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := getClaims(c)
		if !ok {
			respondError(c, http.StatusUnauthorized, "unauthorized", "Claims not found in context")
			c.Abort()
			return
		}

		if !claims.HasPermission(requiredPerm) {
			respondError(c, http.StatusForbidden, "access_denied", "Insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

// respondError - вспомогательная функция для возврата ошибок в едином формате
func respondError(c *gin.Context, status int, code string, msg string) {
	c.AbortWithStatusJSON(status, dto.ErrorResponse{
		Error:   code,
		Message: msg,
	})
}
