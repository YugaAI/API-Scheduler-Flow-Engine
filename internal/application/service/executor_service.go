package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/repository"
	"github.com/openspec/api-scheduler-flow-engine/internal/infrastructure/queue"
	"github.com/openspec/api-scheduler-flow-engine/pkg/config"
	"github.com/openspec/api-scheduler-flow-engine/pkg/logger"
)

type ExecutorService struct {
	executionRepo  repository.ExecutionRepository
	flowRepo       repository.FlowRepository
	actionRegistry *ActionRegistry
	retryTracker   queue.RetryTracker
	cfg            *config.Config
}

func NewExecutorService(
	executionRepo repository.ExecutionRepository,
	flowRepo repository.FlowRepository,
	actionRegistry *ActionRegistry,
	retryTracker queue.RetryTracker,
	cfg *config.Config,
) *ExecutorService {
	return &ExecutorService{
		executionRepo:  executionRepo,
		flowRepo:       flowRepo,
		actionRegistry: actionRegistry,
		retryTracker:   retryTracker,
		cfg:            cfg,
	}
}

func (s *ExecutorService) Execute(ctx context.Context, executionID uuid.UUID) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(s.cfg.ExecutionTimeoutSeconds)*time.Second)
	defer cancel()

	execution, err := s.executionRepo.FindByID(timeoutCtx, executionID)
	if err != nil || execution == nil {
		logger.Error("Failed to find execution", "execution_id", executionID, "error", err)
		return
	}

	if execution.FlowID == nil {
		logger.Error("Execution has no associated flow", "execution_id", executionID)
		s.markExecutionFinished(timeoutCtx, execution, entity.ExecutionStatusFailed)
		return
	}

	flow, err := s.flowRepo.FindByID(timeoutCtx, *execution.FlowID)
	if err != nil || flow == nil {
		logger.Error("Failed to find flow for execution", "execution_id", executionID, "flow_id", execution.FlowID)
		s.markExecutionFinished(timeoutCtx, execution, entity.ExecutionStatusFailed)
		return
	}

	now := time.Now()
	execution.Status = entity.ExecutionStatusRunning
	execution.StartedAt = &now
	s.executionRepo.Update(timeoutCtx, execution)

	logger.Info("Starting execution", "execution_id", executionID, "flow_id", flow.ID, "trigger_type", execution.TriggerType)

	overallStatus := entity.ExecutionStatusCompleted
	stopExecution := false

	for _, step := range flow.Steps {
		if stopExecution {
			s.createSkippedStep(timeoutCtx, executionID, step)
			continue
		}

		err := s.executeStep(timeoutCtx, executionID, step)
		if err != nil {
			overallStatus = entity.ExecutionStatusFailed
			if s.cfg.FailurePolicy == "stop" {
				stopExecution = true
			}
		}
	}

	if timeoutCtx.Err() == context.DeadlineExceeded {
		overallStatus = entity.ExecutionStatusFailed
		logger.Error("Execution timed out", "execution_id", executionID)
	}

	s.markExecutionFinished(context.Background(), execution, overallStatus)
}

func (s *ExecutorService) executeStep(ctx context.Context, executionID uuid.UUID, step entity.Step) error {
	now := time.Now()
	execStep := &entity.ExecutionStep{
		ExecutionID: executionID,
		StepOrder:   step.Order,
		Action:      step.Action,
		Status:      entity.StepStatusRunning,
		StartedAt:   &now,
	}

	s.executionRepo.UpdateStep(ctx, execStep)

	actionHandler, err := s.actionRegistry.Get(step.Action)
	if err != nil {
		execStep.Status = entity.StepStatusFailed
		execStep.Log = fmt.Sprintf("Action not found: %s", err.Error())
		s.finishStep(ctx, execStep)
		return err
	}

	retryCount := step.RetryCount
	retryDelay := time.Duration(step.RetryDelaySeconds) * time.Second

	var output string
	var execErr error

	for attempt := 0; attempt <= retryCount; attempt++ {
		execStep.RetryAttempts = attempt

		output, execErr = actionHandler.Execute(ctx, step.Config)
		if execErr == nil {
			// Step berhasil — jika sebelumnya ada retry state, tandai completed
			if attempt > 0 {
				if trackErr := s.retryTracker.TrackRetryCompleted(ctx, executionID, step.Order); trackErr != nil {
					logger.Warn("Failed to track retry completed",
						"execution_id", executionID,
						"step_order", step.Order,
						"error", trackErr,
					)
				}
			}
			break
		}

		if attempt >= retryCount {
			// Semua retry habis — tandai failed di Redis
			if retryCount > 0 {
				if trackErr := s.retryTracker.TrackRetryFailed(ctx, executionID, step.Order, execErr.Error()); trackErr != nil {
					logger.Warn("Failed to track retry failed",
						"execution_id", executionID,
						"step_order", step.Order,
						"error", trackErr,
					)
				}
			}
			break
		}

		// Masih ada sisa retry — log dan track ke Redis
		nextRetryAt := time.Now().Add(retryDelay)
		logger.Warn("Step execution failed, retrying",
			"execution_id", executionID,
			"step_order", step.Order,
			"action", step.Action,
			"attempt", attempt+1,
			"max_retries", retryCount,
			"next_retry_at", nextRetryAt.Format(time.RFC3339),
			"error", execErr,
		)

		// Track retry attempt ke Redis — visible di RedisInsight
		if trackErr := s.retryTracker.TrackRetryAttempt(
			ctx,
			executionID,
			step.Order,
			attempt+1,
			retryCount,
			step.Action,
			execErr.Error(),
			nextRetryAt,
		); trackErr != nil {
			logger.Warn("Failed to track retry attempt",
				"execution_id", executionID,
				"step_order", step.Order,
				"error", trackErr,
			)
		}

		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			execErr = ctx.Err()
			// Track sebagai failed karena context cancelled
			if retryCount > 0 {
				_ = s.retryTracker.TrackRetryFailed(ctx, executionID, step.Order, execErr.Error())
			}
			goto doneRetry
		case <-timer.C:
			// lanjut ke attempt berikutnya
		}
	}

doneRetry:
	execStep.Log = output
	if execErr != nil {
		execStep.Status = entity.StepStatusFailed
		execStep.Log += fmt.Sprintf("\nError: %s", execErr.Error())
		s.finishStep(ctx, execStep)
		return execErr
	}

	execStep.Status = entity.StepStatusCompleted
	s.finishStep(ctx, execStep)
	return nil
}

func (s *ExecutorService) createSkippedStep(ctx context.Context, executionID uuid.UUID, step entity.Step) {
	execStep := &entity.ExecutionStep{
		ExecutionID: executionID,
		StepOrder:   step.Order,
		Action:      step.Action,
		Status:      entity.StepStatusSkipped,
	}
	s.executionRepo.UpdateStep(ctx, execStep)
}

func (s *ExecutorService) finishStep(ctx context.Context, step *entity.ExecutionStep) {
	now := time.Now()
	step.FinishedAt = &now
	s.executionRepo.UpdateStep(ctx, step)
}

func (s *ExecutorService) markExecutionFinished(ctx context.Context, execution *entity.Execution, status string) {
	now := time.Now()
	execution.Status = status
	execution.FinishedAt = &now
	s.executionRepo.Update(ctx, execution)
	logger.Info("Execution finished", "execution_id", execution.ID, "status", status)
}
