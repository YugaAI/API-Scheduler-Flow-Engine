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

	// FindByID loads execution WITHOUT steps — dipakai di executor path untuk metadata saja.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Execution, error)

	// FindByIDWithSteps loads execution WITH steps preloaded — dipakai di presentation layer.
	FindByIDWithSteps(ctx context.Context, id uuid.UUID) (*entity.Execution, error)

	FindAll(ctx context.Context, filter ExecutionFilter, page, pageSize int, sortBy, sortOrder string) ([]entity.Execution, int64, error)

	// Update hanya meng-update field status, started_at, finished_at — bukan full save.
	Update(ctx context.Context, execution *entity.Execution) error

	// CreateStep insert row baru untuk execution step.
	CreateStep(ctx context.Context, step *entity.ExecutionStep) error

	// UpdateStep hanya meng-update field yang relevan — bukan full save.
	UpdateStep(ctx context.Context, step *entity.ExecutionStep) error
}
