// internal/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vega-trello/trello-back/internal/auth"
	dto "github.com/vega-trello/trello-back/internal/dto/user"
)

const ContextKeyUserUUID = "userUUID"

func Auth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "missing_token",
				Message: "Authorization header is required",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "invalid_token_format",
				Message: "Invalid token format. Expected 'Bearer <token>'",
			})
			return
		}

		token := parts[1]
		userUUID, err := jwtManager.Parse(token)
		if err != nil {
			handleAuthError(c, err)
			return
		}

		c.Set(ContextKeyUserUUID, userUUID)
		c.Next()
	}
}

func handleAuthError(c *gin.Context, err error) {
	var code string
	switch {
	case err == auth.ErrTokenExpired:
		code = "token_expired"
	case err == auth.ErrTokenMalformed:
		code = "malformed_token"
	case err == auth.ErrSignatureInvalid:
		code = "invalid_signature"
	default:
		code = "invalid_token"
	}

	c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{
		Error:   code,
		Message: err.Error(),
	})
}

func GetUserUUID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get(ContextKeyUserUUID)
	if !exists {
		return uuid.Nil, false
	}
	userUUID, ok := val.(uuid.UUID)
	return userUUID, ok
}
