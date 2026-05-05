package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
)

type ExecutionStepResponse struct {
	ID            uuid.UUID  `json:"id"`
	StepOrder     int        `json:"step_order"`
	Action        string     `json:"action"`
	Status        string     `json:"status"`
	Log           string     `json:"log,omitempty"`
	RetryAttempts int        `json:"retry_attempts"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
}

type ExecutionResponse struct {
	ID          uuid.UUID               `json:"id"`
	FlowID      *uuid.UUID              `json:"flow_id"`
	Status      string                  `json:"status"`
	TriggerType string                  `json:"trigger_type"`
	StartedAt   *time.Time              `json:"started_at,omitempty"`
	FinishedAt  *time.Time              `json:"finished_at,omitempty"`
	Steps       []ExecutionStepResponse `json:"steps,omitempty"`
}

func MapExecution(exec *entity.Execution) ExecutionResponse {
	var steps []ExecutionStepResponse
	if exec.Steps != nil {
		steps = make([]ExecutionStepResponse, len(exec.Steps))
		for i, s := range exec.Steps {
			steps[i] = ExecutionStepResponse{
				ID:            s.ID,
				StepOrder:     s.StepOrder,
				Action:        s.Action,
				Status:        s.Status,
				Log:           s.Log,
				RetryAttempts: s.RetryAttempts,
				StartedAt:     s.StartedAt,
				FinishedAt:    s.FinishedAt,
			}
		}
	}

	return ExecutionResponse{
		ID:          exec.ID,
		FlowID:      exec.FlowID,
		Status:      exec.Status,
		TriggerType: exec.TriggerType,
		StartedAt:   exec.StartedAt,
		FinishedAt:  exec.FinishedAt,
		Steps:       steps,
	}
}

func MapExecutions(execs []entity.Execution) []ExecutionResponse {
	res := make([]ExecutionResponse, len(execs))
	for i, e := range execs {
		res[i] = MapExecution(&e)
	}
	return res
}
