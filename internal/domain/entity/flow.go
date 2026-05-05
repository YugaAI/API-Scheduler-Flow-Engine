package entity

import (
	"time"

	"github.com/google/uuid"
)

// Flow represents a workflow template containing an ordered sequence of steps.
type Flow struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null;unique" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	Steps       []Step    `gorm:"foreignKey:FlowID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"steps"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
