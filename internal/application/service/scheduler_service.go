package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/repository"
	"github.com/openspec/api-scheduler-flow-engine/pkg/logger"
	"github.com/robfig/cron/v3"
)

type ExecutionDispatcher interface {
	Dispatch(ctx context.Context, executionID uuid.UUID) error
}

type SchedulerService struct {
	cronRunner    *cron.Cron
	scheduleRepo  repository.ScheduleRepository
	executionRepo repository.ExecutionRepository
	dispatcher    ExecutionDispatcher
	timezone      *time.Location

	mu   sync.RWMutex
	jobs map[uuid.UUID]cron.EntryID
}

func NewSchedulerService(
	scheduleRepo repository.ScheduleRepository,
	executionRepo repository.ExecutionRepository,
	dispatcher ExecutionDispatcher,
	timezoneStr string,
) (*SchedulerService, error) {
	loc, err := time.LoadLocation(timezoneStr)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}

	cronRunner := cron.New(cron.WithLocation(loc))

	return &SchedulerService{
		cronRunner:    cronRunner,
		scheduleRepo:  scheduleRepo,
		executionRepo: executionRepo,
		dispatcher:    dispatcher,
		timezone:      loc,
		jobs:          make(map[uuid.UUID]cron.EntryID),
	}, nil
}

func (s *SchedulerService) Start(ctx context.Context) error {
	logger.Info("Starting scheduler service")

	// Reload all enabled schedules from DB
	schedules, err := s.scheduleRepo.FindAllEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to load enabled schedules: %w", err)
	}

	for _, sched := range schedules {
		err := s.AddJob(sched.ID, sched.CronExpression, sched.FlowID)
		if err != nil {
			logger.Error("Failed to register schedule on startup", "schedule_id", sched.ID, "error", err)
		}
	}

	s.cronRunner.Start()
	logger.Info("Scheduler service started", "loaded_jobs", len(schedules))
	return nil
}

func (s *SchedulerService) Stop() {
	logger.Info("Stopping scheduler service")
	s.cronRunner.Stop()
}

func (s *SchedulerService) AddJob(scheduleID uuid.UUID, cronExpr string, flowID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing job if present
	if entryID, exists := s.jobs[scheduleID]; exists {
		s.cronRunner.Remove(entryID)
	}

	job := func() {
		logger.Info("Cron job triggered", "schedule_id", scheduleID, "flow_id", flowID)
		ctx := context.Background()
		
		// Create execution
		execution := &entity.Execution{
			FlowID:      &flowID,
			Status:      entity.ExecutionStatusPending,
			TriggerType: entity.TriggerTypeScheduled,
		}

		if err := s.executionRepo.Create(ctx, execution); err != nil {
			logger.Error("Failed to create execution for cron trigger", "error", err)
			return
		}

		// Dispatch execution
		if err := s.dispatcher.Dispatch(ctx, execution.ID); err != nil {
			logger.Error("Failed to dispatch scheduled execution", "execution_id", execution.ID, "error", err)
			return
		}

		// Update schedule last_run_at
		now := time.Now()
		sched, err := s.scheduleRepo.FindByID(ctx, scheduleID)
		if err == nil && sched != nil {
			sched.LastRunAt = &now
			s.scheduleRepo.Update(ctx, sched)
		}
	}

	entryID, err := s.cronRunner.AddFunc(cronExpr, job)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.jobs[scheduleID] = entryID
	logger.Debug("Cron job added", "schedule_id", scheduleID, "cron_expr", cronExpr)
	return nil
}

func (s *SchedulerService) RemoveJob(scheduleID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.jobs[scheduleID]; exists {
		s.cronRunner.Remove(entryID)
		delete(s.jobs, scheduleID)
		logger.Debug("Cron job removed", "schedule_id", scheduleID)
	}
}
