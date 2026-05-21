package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/repository"
	infraAction "github.com/openspec/api-scheduler-flow-engine/internal/infrastructure/action"
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

	// FindByID tanpa steps — executor hanya butuh metadata execution
	execution, err := s.executionRepo.FindByID(timeoutCtx, executionID)
	if err != nil || execution == nil {
		logger.Error("Failed to find execution", "execution_id", executionID, "error", err)
		return
	}

	if execution.FlowID == nil {
		logger.Error("Execution has no associated flow", "execution_id", executionID)
		s.markExecutionFinished(executionID, entity.ExecutionStatusFailed)
		return
	}

	flow, err := s.flowRepo.FindByID(timeoutCtx, *execution.FlowID)
	if err != nil || flow == nil {
		logger.Error("Failed to find flow for execution", "execution_id", executionID, "flow_id", execution.FlowID)
		s.markExecutionFinished(executionID, entity.ExecutionStatusFailed)
		return
	}

	now := time.Now()
	execution.Status = entity.ExecutionStatusRunning
	execution.StartedAt = &now
	if err := s.executionRepo.Update(timeoutCtx, execution); err != nil {
		logger.Error("Failed to update execution status to running", "execution_id", executionID, "error", err)
	}

	logger.Info("Starting execution",
		"execution_id", executionID,
		"flow_id", flow.ID,
		"trigger_type", execution.TriggerType,
	)

	overallStatus := entity.ExecutionStatusCompleted
	stopExecution := false

	for _, step := range flow.Steps {
		if stopExecution {
			s.createSkippedStep(timeoutCtx, executionID, step)
			continue
		}

		if err := s.executeStep(timeoutCtx, executionID, step); err != nil {
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

	// markExecutionFinished memakai context baru dengan timeout sendiri —
	// timeoutCtx bisa sudah expired saat ini
	s.markExecutionFinished(executionID, overallStatus)
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

	// CreateStep — INSERT row baru, sehingga execStep.ID terisi oleh GORM.
	// Semua UpdateStep berikutnya aman karena sudah ada ID.
	if err := s.executionRepo.CreateStep(ctx, execStep); err != nil {
		logger.Error("Failed to create execution step",
			"execution_id", executionID,
			"step_order", step.Order,
			"error", err,
		)
		return err
	}

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
			// Step berhasil — jika sebelumnya ada retry, tandai completed di Redis
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

		// Cek retryability SEBELUM melakukan apapun —
		// non-retryable error tidak boleh masuk ke delay/track loop
		if !infraAction.IsRetryable(execErr) {
			logger.Error("Step failed with non-retryable error, aborting retries",
				"execution_id", executionID,
				"step_order", step.Order,
				"action", step.Action,
				"attempt", attempt+1,
				"error", execErr,
			)

			// Track ke Redis sebagai failed langsung jika retryCount > 0
			if retryCount > 0 {
				trackCtx, trackCancel := context.WithTimeout(context.Background(), 3*time.Second)
				if trackErr := s.retryTracker.TrackRetryFailed(trackCtx, executionID, step.Order, execErr.Error()); trackErr != nil {
					logger.Warn("Failed to track non-retryable failure",
						"execution_id", executionID,
						"step_order", step.Order,
						"error", trackErr,
					)
				}
				trackCancel()
			}
			break
		}

		if attempt >= retryCount {
			// Semua retry habis
			if retryCount > 0 {
				trackCtx, trackCancel := context.WithTimeout(context.Background(), 3*time.Second)
				if trackErr := s.retryTracker.TrackRetryFailed(trackCtx, executionID, step.Order, execErr.Error()); trackErr != nil {
					logger.Warn("Failed to track retry exhausted",
						"execution_id", executionID,
						"step_order", step.Order,
						"error", trackErr,
					)
				}
				trackCancel()
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

			// ctx sudah done — gunakan context baru untuk Redis tracking
			if retryCount > 0 {
				trackCtx, trackCancel := context.WithTimeout(context.Background(), 3*time.Second)
				if trackErr := s.retryTracker.TrackRetryFailed(trackCtx, executionID, step.Order, execErr.Error()); trackErr != nil {
					logger.Warn("Failed to track retry cancelled",
						"execution_id", executionID,
						"step_order", step.Order,
						"error", trackErr,
					)
				}
				trackCancel()
			}
			goto doneRetry

		case <-timer.C:
			// Lanjut ke attempt berikutnya
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
	if err := s.executionRepo.CreateStep(ctx, execStep); err != nil {
		logger.Error("Failed to create skipped step record",
			"execution_id", executionID,
			"step_order", step.Order,
			"error", err,
		)
	}
}

func (s *ExecutorService) finishStep(ctx context.Context, step *entity.ExecutionStep) {
	now := time.Now()
	step.FinishedAt = &now
	if err := s.executionRepo.UpdateStep(ctx, step); err != nil {
		logger.Error("Failed to update step status",
			"step_id", step.ID,
			"execution_id", step.ExecutionID,
			"step_order", step.StepOrder,
			"status", step.Status,
			"error", err,
		)
	}
}

// markExecutionFinished selalu menggunakan context baru dengan timeout sendiri.
// Dipanggil setelah timeoutCtx bisa sudah expired — tidak boleh pakai ctx lama.
func (s *ExecutorService) markExecutionFinished(executionID uuid.UUID, status string) {
	finishCtx, finishCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer finishCancel()

	now := time.Now()
	execution := &entity.Execution{
		ID:         executionID,
		Status:     status,
		FinishedAt: &now,
	}
	if err := s.executionRepo.Update(finishCtx, execution); err != nil {
		logger.Error("Failed to mark execution finished",
			"execution_id", executionID,
			"status", status,
			"error", err,
		)
		return
	}

	logger.Info("Execution finished", "execution_id", executionID, "status", status)
}
