package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/application/usecase"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/dto/request"
	"github.com/openspec/api-scheduler-flow-engine/internal/presentation/dto/response"
)

type ScheduleHandler struct {
	usecase usecase.ScheduleUseCase
}

func NewScheduleHandler(u usecase.ScheduleUseCase) *ScheduleHandler {
	return &ScheduleHandler{usecase: u}
}

func (h *ScheduleHandler) CreateSchedule(c *gin.Context) {
	flowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid flow ID format"})
		return
	}

	var req request.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid request body", Details: err.Error()})
		return
	}

	schedule := &entity.Schedule{
		FlowID:         flowID,
		CronExpression: req.CronExpression,
	}

	if err := h.usecase.CreateSchedule(c.Request.Context(), schedule); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response.MapSchedule(schedule))
}

func (h *ScheduleHandler) UpdateSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid schedule ID format"})
		return
	}

	var req request.UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid request body", Details: err.Error()})
		return
	}

	var schedule *entity.Schedule
	if req.Enabled {
		schedule, err = h.usecase.EnableSchedule(c.Request.Context(), id)
	} else {
		schedule, err = h.usecase.DisableSchedule(c.Request.Context(), id)
	}

	if err != nil {
		if err.Error() == "schedule not found" {
			c.JSON(http.StatusNotFound, response.ErrorResponse{Error: err.Error()})
			return
		}
		c.Error(err)
		c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response.MapSchedule(schedule))
}
