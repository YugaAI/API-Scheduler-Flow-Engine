package router

import (
	"github.com/gin-gonic/gin"
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/handler"
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/middleware"
)

func SetupRouter(
	jwtSecret string,
	flowHandler *handler.FlowHandler,
	executionHandler *handler.ExecutionHandler,
	scheduleHandler *handler.ScheduleHandler,
) *gin.Engine {
	r := gin.Default()

	r.Use(middleware.ErrorHandler())

	v1 := r.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(jwtSecret))

	// Flows
	flows := v1.Group("/flows")
	{
		flows.GET("", middleware.RoleMiddleware("ADMIN", "USER"), flowHandler.ListFlows)
		flows.GET("/:id", middleware.RoleMiddleware("ADMIN", "USER"), flowHandler.GetFlow)
		flows.POST("", middleware.RoleMiddleware("ADMIN"), flowHandler.CreateFlow)
		flows.PUT("/:id", middleware.RoleMiddleware("ADMIN"), flowHandler.UpdateFlow)
		flows.DELETE("/:id", middleware.RoleMiddleware("ADMIN"), flowHandler.DeleteFlow)

		flows.POST("/:id/execute", middleware.RoleMiddleware("ADMIN", "USER"), executionHandler.TriggerExecution)
		
		flows.POST("/:id/schedule", middleware.RoleMiddleware("ADMIN"), scheduleHandler.CreateSchedule)
	}

	// Executions
	executions := v1.Group("/executions")
	{
		executions.GET("", middleware.RoleMiddleware("ADMIN", "USER"), executionHandler.ListExecutions)
		executions.GET("/:id", middleware.RoleMiddleware("ADMIN", "USER"), executionHandler.GetExecution)
	}

	// Schedules
	schedules := v1.Group("/schedules")
	{
		schedules.PATCH("/:id", middleware.RoleMiddleware("ADMIN"), scheduleHandler.UpdateSchedule)
	}

	return r
}
