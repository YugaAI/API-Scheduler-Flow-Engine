package entity

import (
	"time"

	"github.com/google/uuid"
)

// Schedule represents a cron schedule for automatically triggering a flow.
type Schedule struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	FlowID         uuid.UUID  `gorm:"type:uuid;not null;index;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"flow_id"`
	CronExpression string     `gorm:"type:varchar(100);not null" json:"cron_expression"`
	Enabled        bool       `gorm:"not null;default:true" json:"enabled"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
