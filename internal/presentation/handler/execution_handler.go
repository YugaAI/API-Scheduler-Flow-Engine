package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/application/usecase"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/repository"
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/dto/response"
)

type ExecutionHandler struct {
	usecase usecase.ExecutionUseCase
}

func NewExecutionHandler(u usecase.ExecutionUseCase) *ExecutionHandler {
	return &ExecutionHandler{usecase: u}
}

func (h *ExecutionHandler) TriggerExecution(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid flow ID format"})
		return
	}

	execution, err := h.usecase.TriggerExecution(c.Request.Context(), id, entity.TriggerTypeManual)
	if err != nil {
		if err.Error() == "flow not found" {
			c.JSON(http.StatusNotFound, response.ErrorResponse{Error: err.Error()})
			return
		}
		c.Error(err)
		return
	}

	c.JSON(http.StatusAccepted, response.MapExecution(execution))
}

func (h *ExecutionHandler) GetExecution(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid execution ID format"})
		return
	}

	// GetExecution dari usecase sudah pakai FindByIDWithSteps — steps ikut di-load
	execution, err := h.usecase.GetExecution(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "execution not found" {
			c.JSON(http.StatusNotFound, response.ErrorResponse{Error: err.Error()})
			return
		}
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response.MapExecution(execution))
}

func (h *ExecutionHandler) ListExecutions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	sortBy := c.DefaultQuery("sort", "started_at")
	sortOrder := c.DefaultQuery("order", "desc")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Validasi status — tolak value yang tidak valid sebelum masuk ke repo
	validStatuses := map[string]bool{
		"pending": true, "running": true,
		"completed": true, "failed": true,
		"cancelled": true, "": true,
	}
	statusFilter := c.Query("status")
	if !validStatuses[statusFilter] {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{
			Error: "invalid status filter, allowed: pending, running, completed, failed, cancelled",
		})
		return
	}

	var filter repository.ExecutionFilter
	if flowIDStr := c.Query("flow_id"); flowIDStr != "" {
		parsed, err := uuid.Parse(flowIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid flow_id format"})
			return
		}
		filter.FlowID = &parsed
	}
	filter.Status = statusFilter

	executions, total, err := h.usecase.ListExecutions(c.Request.Context(), filter, page, pageSize, sortBy, sortOrder)
	if err != nil {
		c.Error(err)
		return
	}

	res := response.NewPaginatedResponse(response.MapExecutions(executions), total, page, pageSize)
	c.JSON(http.StatusOK, res)
}
