package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
)

type ScheduleResponse struct {
	ID             uuid.UUID  `json:"id"`
	FlowID         uuid.UUID  `json:"flow_id"`
	CronExpression string     `json:"cron_expression"`
	Enabled        bool       `json:"enabled"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func MapSchedule(sched *entity.Schedule) ScheduleResponse {
	return ScheduleResponse{
		ID:             sched.ID,
		FlowID:         sched.FlowID,
		CronExpression: sched.CronExpression,
		Enabled:        sched.Enabled,
		LastRunAt:      sched.LastRunAt,
		CreatedAt:      sched.CreatedAt,
	}
}
