package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/application/usecase"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/dto/request"
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/dto/response"
)

type FlowHandler struct {
	usecase usecase.FlowUseCase
}

func NewFlowHandler(u usecase.FlowUseCase) *FlowHandler {
	return &FlowHandler{usecase: u}
}

func (h *FlowHandler) CreateFlow(c *gin.Context) {
	var req request.CreateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid request body", Details: err.Error()})
		return
	}

	flow := &entity.Flow{
		Name:        req.Name,
		Description: req.Description,
		Steps:       mapStepRequestsToEntities(req.Steps),
	}

	if err := h.usecase.CreateFlow(c.Request.Context(), flow); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response.MapFlow(flow))
}

func (h *FlowHandler) GetFlow(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid flow ID format"})
		return
	}

	flow, err := h.usecase.GetFlow(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "flow not found" {
			c.JSON(http.StatusNotFound, response.ErrorResponse{Error: err.Error()})
			return
		}
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response.MapFlow(flow))
}

func (h *FlowHandler) ListFlows(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	flows, total, err := h.usecase.ListFlows(c.Request.Context(), page, pageSize)
	if err != nil {
		c.Error(err)
		return
	}

	res := response.NewPaginatedResponse(response.MapFlows(flows), total, page, pageSize)
	c.JSON(http.StatusOK, res)
}

func (h *FlowHandler) UpdateFlow(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid flow ID format"})
		return
	}

	var req request.UpdateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid request body", Details: err.Error()})
		return
	}

	flow := &entity.Flow{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Steps:       mapStepRequestsToEntities(req.Steps),
	}

	if err := h.usecase.UpdateFlow(c.Request.Context(), flow); err != nil {
		if err.Error() == "flow not found" {
			c.JSON(http.StatusNotFound, response.ErrorResponse{Error: err.Error()})
			return
		}
		c.Error(err)
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
		return
	}

	// Fetch updated flow to return complete data
	updatedFlow, err := h.usecase.GetFlow(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response.MapFlow(updatedFlow))
}

func (h *FlowHandler) DeleteFlow(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid flow ID format"})
		return
	}

	if err := h.usecase.DeleteFlow(c.Request.Context(), id); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func mapStepRequestsToEntities(reqs []request.StepRequest) []entity.Step {
	steps := make([]entity.Step, len(reqs))
	for i, r := range reqs {
		steps[i] = entity.Step{
			Order:  r.Order,
			Action: r.Action,
			Config: r.Config,
		}
		if r.RetryCount != nil {
			steps[i].RetryCount = *r.RetryCount
		} else {
			steps[i].RetryCount = 3 // default
		}
		if r.RetryDelaySeconds != nil {
			steps[i].RetryDelaySeconds = *r.RetryDelaySeconds
		} else {
			steps[i].RetryDelaySeconds = 5 // default
		}
	}
	return steps
}
