package response

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
)

type StepResponse struct {
	ID                uuid.UUID       `json:"id"`
	Order             int             `json:"order"`
	Action            string          `json:"action"`
	Config            json.RawMessage `json:"config"`
	RetryCount        int             `json:"retry_count"`
	RetryDelaySeconds int             `json:"retry_delay_seconds"`
}

type FlowResponse struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Steps       []StepResponse `json:"steps"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func MapFlow(flow *entity.Flow) FlowResponse {
	steps := make([]StepResponse, len(flow.Steps))
	for i, s := range flow.Steps {
		steps[i] = StepResponse{
			ID:                s.ID,
			Order:             s.Order,
			Action:            s.Action,
			Config:            s.Config,
			RetryCount:        s.RetryCount,
			RetryDelaySeconds: s.RetryDelaySeconds,
		}
	}

	return FlowResponse{
		ID:          flow.ID,
		Name:        flow.Name,
		Description: flow.Description,
		Steps:       steps,
		CreatedAt:   flow.CreatedAt,
		UpdatedAt:   flow.UpdatedAt,
	}
}

func MapFlows(flows []entity.Flow) []FlowResponse {
	res := make([]FlowResponse, len(flows))
	for i, f := range flows {
		res[i] = MapFlow(&f)
	}
	return res
}
