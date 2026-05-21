package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	ExecutionStatusPending   = "pending"
	ExecutionStatusRunning   = "running"
	ExecutionStatusCompleted = "completed"
	ExecutionStatusFailed    = "failed"
	ExecutionStatusCancelled = "cancelled"

	TriggerTypeManual    = "manual"
	TriggerTypeScheduled = "scheduled"
)

// Execution represents an instance of a flow being executed.
type Execution struct {
	ID          uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	FlowID      *uuid.UUID      `gorm:"type:uuid;index;constraint:OnDelete:SET NULL" json:"flow_id"`
	Status      string          `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	TriggerType string          `gorm:"type:varchar(20);not null" json:"trigger_type"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	FinishedAt  *time.Time      `json:"finished_at,omitempty"`
	CreatedAt   time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
	Steps       []ExecutionStep `gorm:"foreignKey:ExecutionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"steps,omitempty"`
}
