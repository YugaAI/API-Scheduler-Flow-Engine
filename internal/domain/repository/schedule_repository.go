package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
)

// ScheduleRepository defines the interface for interacting with Schedule data.
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *entity.Schedule) error
	FindByFlowID(ctx context.Context, flowID uuid.UUID) (*entity.Schedule, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Schedule, error)
	FindAllEnabled(ctx context.Context) ([]entity.Schedule, error)
	Update(ctx context.Context, schedule *entity.Schedule) error
	Delete(ctx context.Context, id uuid.UUID) error
}
