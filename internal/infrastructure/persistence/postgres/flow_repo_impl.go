package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/repository"
	"gorm.io/gorm"
)

type flowRepositoryImpl struct {
	db *gorm.DB
}

// NewFlowRepository creates a new instance of FlowRepository using GORM.
func NewFlowRepository(db *gorm.DB) repository.FlowRepository {
	return &flowRepositoryImpl{db: db}
}

func (r *flowRepositoryImpl) Create(ctx context.Context, flow *entity.Flow) error {
	return r.db.WithContext(ctx).Create(flow).Error
}

func (r *flowRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entity.Flow, error) {
	var flow entity.Flow
	err := r.db.WithContext(ctx).Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("steps.order ASC")
	}).First(&flow, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // or define a custom domain error
		}
		return nil, err
	}
	return &flow, nil
}

func (r *flowRepositoryImpl) FindAll(ctx context.Context, page, pageSize int) ([]entity.Flow, int64, error) {
	var flows []entity.Flow
	var total int64

	// Count total records
	if err := r.db.WithContext(ctx).Model(&entity.Flow{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&flows).Error

	if err != nil {
		return nil, 0, err
	}

	return flows, total, nil
}

func (r *flowRepositoryImpl) Update(ctx context.Context, flow *entity.Flow) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update flow fields
		if err := tx.Save(flow).Error; err != nil {
			return err
		}

		// Update steps (full replacement strategy: delete existing steps and insert new ones)
		if err := tx.Where("flow_id = ?", flow.ID).Delete(&entity.Step{}).Error; err != nil {
			return err
		}

		if len(flow.Steps) > 0 {
			if err := tx.Create(&flow.Steps).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *flowRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	// Cascade delete handles steps and schedules due to constraints
	return r.db.WithContext(ctx).Delete(&entity.Flow{}, "id = ?", id).Error
}
