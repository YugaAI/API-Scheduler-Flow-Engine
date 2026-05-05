package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
)

// ExecutionFilter defines optional filters for listing executions.
type ExecutionFilter struct {
	FlowID *uuid.UUID
	Status string
}

// ExecutionRepository defines the interface for interacting with Execution data.
type ExecutionRepository interface {
	Create(ctx context.Context, execution *entity.Execution) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Execution, error)
	FindAll(ctx context.Context, filter ExecutionFilter, page, pageSize int, sortBy, sortOrder string) ([]entity.Execution, int64, error)
	Update(ctx context.Context, execution *entity.Execution) error
	UpdateStep(ctx context.Context, step *entity.ExecutionStep) error
}
