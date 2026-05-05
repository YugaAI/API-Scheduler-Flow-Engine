package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/dto/response"
	"github.com/openspec/api-scheduler-flow-engine/pkg/logger"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			logger.Error("Request error", "path", c.Request.URL.Path, "error", err.Err)

			// Here you could add logic to map specific domain errors to HTTP status codes.
			// For simplicity, we just return a 500 or 400 based on context if not already set.
			
			status := c.Writer.Status()
			if status == 200 { // If no status was set explicitly
				status = http.StatusInternalServerError
			}

			c.JSON(status, response.ErrorResponse{
				Error:   err.Error(),
			})
		}
	}
}
