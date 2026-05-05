package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/repository"
	"gorm.io/gorm"
)

type executionRepositoryImpl struct {
	db *gorm.DB
}

// NewExecutionRepository creates a new instance of ExecutionRepository using GORM.
func NewExecutionRepository(db *gorm.DB) repository.ExecutionRepository {
	return &executionRepositoryImpl{db: db}
}

func (r *executionRepositoryImpl) Create(ctx context.Context, execution *entity.Execution) error {
	return r.db.WithContext(ctx).Create(execution).Error
}

func (r *executionRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entity.Execution, error) {
	var execution entity.Execution
	err := r.db.WithContext(ctx).Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("execution_steps.step_order ASC")
	}).First(&execution, "id = ?", id).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &execution, nil
}

func (r *executionRepositoryImpl) FindAll(ctx context.Context, filter repository.ExecutionFilter, page, pageSize int, sortBy, sortOrder string) ([]entity.Execution, int64, error) {
	var executions []entity.Execution
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Execution{})

	// Apply filters
	if filter.FlowID != nil {
		query = query.Where("flow_id = ?", *filter.FlowID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	if sortBy == "" {
		sortBy = "started_at"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}
	orderClause := fmt.Sprintf("%s %s", sortBy, sortOrder)

	// Apply pagination
	offset := (page - 1) * pageSize
	err := query.Order(orderClause).Offset(offset).Limit(pageSize).Find(&executions).Error
	if err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}

func (r *executionRepositoryImpl) Update(ctx context.Context, execution *entity.Execution) error {
	// gorm Save updates all fields
	return r.db.WithContext(ctx).Save(execution).Error
}

func (r *executionRepositoryImpl) UpdateStep(ctx context.Context, step *entity.ExecutionStep) error {
	return r.db.WithContext(ctx).Save(step).Error
}
