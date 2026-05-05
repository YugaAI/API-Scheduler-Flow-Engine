package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
)

// FlowRepository defines the interface for interacting with Flow data.
type FlowRepository interface {
	Create(ctx context.Context, flow *entity.Flow) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Flow, error)
	FindAll(ctx context.Context, page, pageSize int) ([]entity.Flow, int64, error)
	Update(ctx context.Context, flow *entity.Flow) error
	Delete(ctx context.Context, id uuid.UUID) error
}
