package request

import "encoding/json"

type StepRequest struct {
	Order             int             `json:"order" binding:"required,min=1"`
	Action            string          `json:"action" binding:"required"`
	Config            json.RawMessage `json:"config" binding:"required"`
	RetryCount        *int            `json:"retry_count"`
	RetryDelaySeconds *int            `json:"retry_delay_seconds"`
}

type CreateFlowRequest struct {
	Name        string        `json:"name" binding:"required,max=255"`
	Description string        `json:"description" binding:"max=1000"`
	Steps       []StepRequest `json:"steps" binding:"required,min=1"`
}

type UpdateFlowRequest struct {
	Name        string        `json:"name" binding:"required,max=255"`
	Description string        `json:"description" binding:"max=1000"`
	Steps       []StepRequest `json:"steps" binding:"required,min=1"`
}
