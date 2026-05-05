package entity

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Step represents a single action within a Flow.
type Step struct {
	ID                uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	FlowID            uuid.UUID       `gorm:"type:uuid;not null;index" json:"flow_id"`
	Order             int             `gorm:"not null" json:"order"`
	Action            string          `gorm:"type:varchar(100);not null" json:"action"`
	Config            json.RawMessage `gorm:"type:jsonb" json:"config"`
	RetryCount        int             `gorm:"not null;default:3" json:"retry_count"`
	RetryDelaySeconds int             `gorm:"not null;default:5" json:"retry_delay_seconds"`
}
