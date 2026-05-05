package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/repository"
	"gorm.io/gorm"
)

type scheduleRepositoryImpl struct {
	db *gorm.DB
}

// NewScheduleRepository creates a new instance of ScheduleRepository using GORM.
func NewScheduleRepository(db *gorm.DB) repository.ScheduleRepository {
	return &scheduleRepositoryImpl{db: db}
}

func (r *scheduleRepositoryImpl) Create(ctx context.Context, schedule *entity.Schedule) error {
	return r.db.WithContext(ctx).Create(schedule).Error
}

func (r *scheduleRepositoryImpl) FindByFlowID(ctx context.Context, flowID uuid.UUID) (*entity.Schedule, error) {
	var schedule entity.Schedule
	err := r.db.WithContext(ctx).Where("flow_id = ?", flowID).First(&schedule).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &schedule, nil
}

func (r *scheduleRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entity.Schedule, error) {
	var schedule entity.Schedule
	err := r.db.WithContext(ctx).First(&schedule, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &schedule, nil
}

func (r *scheduleRepositoryImpl) FindAllEnabled(ctx context.Context) ([]entity.Schedule, error) {
	var schedules []entity.Schedule
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&schedules).Error
	return schedules, err
}

func (r *scheduleRepositoryImpl) Update(ctx context.Context, schedule *entity.Schedule) error {
	return r.db.WithContext(ctx).Save(schedule).Error
}

func (r *scheduleRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.Schedule{}, "id = ?", id).Error
}
