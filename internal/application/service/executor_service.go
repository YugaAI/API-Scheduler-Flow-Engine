package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/repository"
	"github.com/openspec/api-scheduler-flow-engine/pkg/config"
	"github.com/openspec/api-scheduler-flow-engine/pkg/logger"
)

type ExecutorService struct {
	executionRepo  repository.ExecutionRepository
	flowRepo       repository.FlowRepository
	actionRegistry *ActionRegistry
	cfg            *config.Config
}

func NewExecutorService(
	executionRepo repository.ExecutionRepository,
	flowRepo repository.FlowRepository,
	actionRegistry *ActionRegistry,
	cfg *config.Config,
) *ExecutorService {
	return &ExecutorService{
		executionRepo:  executionRepo,
		flowRepo:       flowRepo,
		actionRegistry: actionRegistry,
		cfg:            cfg,
	}
}

func (s *ExecutorService) Execute(ctx context.Context, executionID uuid.UUID) {
	// Add timeout context
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

	// Check if context timed out
	if timeoutCtx.Err() == context.DeadlineExceeded {
		overallStatus = entity.ExecutionStatusFailed
		logger.Error("Execution timed out", "execution_id", executionID)
	}

	s.markExecutionFinished(context.Background(), execution, overallStatus) // use background ctx to ensure save
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

	// Initialize step record
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
			break // Success
		}

		// Failed, log and wait if we have retries left
		if attempt < retryCount {
			logger.Warn("Step execution failed, retrying", 
				"execution_id", executionID, 
				"step", step.Order, 
				"attempt", attempt+1, 
				"error", execErr)
			
			select {
			case <-ctx.Done():
				execErr = ctx.Err()
				break
			case <-time.After(retryDelay):
				// retry
			}
		}
	}

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
