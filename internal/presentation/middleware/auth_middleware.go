package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/dto/response"
	"github.com/openspec/api-scheduler-flow-engine/pkg/logger"
)

const (
	ContextUserIDKey = "user_id"
	ContextRoleKey   = "user_role"
)

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse{Error: "authentication required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse{Error: "invalid authorization header format"})
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			logger.Warn("Invalid token", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse{Error: "invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse{Error: "invalid token claims"})
			return
		}

		sub, ok := claims["sub"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse{Error: "missing sub claim"})
			return
		}

		role, ok := claims["role"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.ErrorResponse{Error: "missing role claim"})
			return
		}

		c.Set(ContextUserIDKey, sub)
		c.Set(ContextRoleKey, role)
		c.Next()
	}
}
