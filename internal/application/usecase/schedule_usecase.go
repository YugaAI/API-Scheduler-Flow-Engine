package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/repository"
)

type CronManager interface {
	AddJob(scheduleID uuid.UUID, cronExpr string, flowID uuid.UUID) error
	RemoveJob(scheduleID uuid.UUID)
}

type ScheduleUseCase interface {
	CreateSchedule(ctx context.Context, schedule *entity.Schedule) error
	EnableSchedule(ctx context.Context, id uuid.UUID) (*entity.Schedule, error)
	DisableSchedule(ctx context.Context, id uuid.UUID) (*entity.Schedule, error)
}

type scheduleUseCaseImpl struct {
	scheduleRepo repository.ScheduleRepository
	flowRepo     repository.FlowRepository
	cronManager  CronManager
}

func NewScheduleUseCase(
	scheduleRepo repository.ScheduleRepository,
	flowRepo repository.FlowRepository,
	cronManager CronManager,
) ScheduleUseCase {
	return &scheduleUseCaseImpl{
		scheduleRepo: scheduleRepo,
		flowRepo:     flowRepo,
		cronManager:  cronManager,
	}
}

func (u *scheduleUseCaseImpl) CreateSchedule(ctx context.Context, schedule *entity.Schedule) error {
	flow, err := u.flowRepo.FindByID(ctx, schedule.FlowID)
	if err != nil {
		return err
	}
	if flow == nil {
		return fmt.Errorf("flow not found")
	}

	// Check if already has schedule
	existing, err := u.scheduleRepo.FindByFlowID(ctx, schedule.FlowID)
	if err != nil {
		return err
	}

	if existing != nil {
		// Update existing
		existing.CronExpression = schedule.CronExpression
		existing.Enabled = true
		if err := u.scheduleRepo.Update(ctx, existing); err != nil {
			return err
		}
		schedule.ID = existing.ID
		u.cronManager.RemoveJob(existing.ID)
	} else {
		schedule.Enabled = true
		if err := u.scheduleRepo.Create(ctx, schedule); err != nil {
			return err
		}
	}

	return u.cronManager.AddJob(schedule.ID, schedule.CronExpression, schedule.FlowID)
}

func (u *scheduleUseCaseImpl) EnableSchedule(ctx context.Context, id uuid.UUID) (*entity.Schedule, error) {
	schedule, err := u.scheduleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, fmt.Errorf("schedule not found")
	}

	schedule.Enabled = true
	if err := u.scheduleRepo.Update(ctx, schedule); err != nil {
		return nil, err
	}

	err = u.cronManager.AddJob(schedule.ID, schedule.CronExpression, schedule.FlowID)
	if err != nil {
		return nil, err
	}

	return schedule, nil
}

func (u *scheduleUseCaseImpl) DisableSchedule(ctx context.Context, id uuid.UUID) (*entity.Schedule, error) {
	schedule, err := u.scheduleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, fmt.Errorf("schedule not found")
	}

	schedule.Enabled = false
	if err := u.scheduleRepo.Update(ctx, schedule); err != nil {
		return nil, err
	}

	u.cronManager.RemoveJob(schedule.ID)
	return schedule, nil
}
