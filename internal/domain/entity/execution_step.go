package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	StepStatusPending   = "pending"
	StepStatusRunning   = "running"
	StepStatusCompleted = "completed"
	StepStatusFailed    = "failed"
	StepStatusSkipped   = "skipped"
)

// ExecutionStep represents the result of a single step execution.
type ExecutionStep struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ExecutionID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"execution_id"`
	StepOrder     int        `gorm:"not null" json:"step_order"`
	Action        string     `gorm:"type:varchar(100);not null" json:"action"`
	Status        string     `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	Log           string     `gorm:"type:text" json:"log"`
	RetryAttempts int        `gorm:"not null;default:0" json:"retry_attempts"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
}
