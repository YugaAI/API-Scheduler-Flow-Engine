package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/dto/response"
)

// RoleMiddleware enforces that the authenticated user has one of the allowed roles.
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(ContextRoleKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse{Error: "authentication required"})
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.ErrorResponse{Error: "invalid role data"})
			return
		}

		isAllowed := false
		for _, role := range allowedRoles {
			if roleStr == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.AbortWithStatusJSON(http.StatusForbidden, response.ErrorResponse{Error: "insufficient permissions"})
			return
		}

		c.Next()
	}
}
