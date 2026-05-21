package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/repository"
)

type ExecutionDispatcher interface {
	Dispatch(ctx context.Context, executionID uuid.UUID) error
}

type ExecutionUseCase interface {
	TriggerExecution(ctx context.Context, flowID uuid.UUID, triggerType string) (*entity.Execution, error)
	GetExecution(ctx context.Context, id uuid.UUID) (*entity.Execution, error)
	ListExecutions(ctx context.Context, filter repository.ExecutionFilter, page, pageSize int, sortBy, sortOrder string) ([]entity.Execution, int64, error)
}

type executionUseCaseImpl struct {
	executionRepo repository.ExecutionRepository
	flowRepo      repository.FlowRepository
	dispatcher    ExecutionDispatcher
}

func NewExecutionUseCase(
	executionRepo repository.ExecutionRepository,
	flowRepo repository.FlowRepository,
	dispatcher ExecutionDispatcher,
) ExecutionUseCase {
	return &executionUseCaseImpl{
		executionRepo: executionRepo,
		flowRepo:      flowRepo,
		dispatcher:    dispatcher,
	}
}

func (u *executionUseCaseImpl) TriggerExecution(ctx context.Context, flowID uuid.UUID, triggerType string) (*entity.Execution, error) {
	flow, err := u.flowRepo.FindByID(ctx, flowID)
	if err != nil {
		return nil, err
	}
	if flow == nil {
		return nil, fmt.Errorf("flow not found")
	}

	execution := &entity.Execution{
		FlowID:      &flowID,
		Status:      entity.ExecutionStatusPending,
		TriggerType: triggerType,
	}

	if err := u.executionRepo.Create(ctx, execution); err != nil {
		return nil, err
	}

	// Dispatch menggunakan ctx dari request — bukan context.Background().
	// Jika request di-cancel sebelum dispatch selesai, operasi ikut di-cancel.
	if err := u.dispatcher.Dispatch(ctx, execution.ID); err != nil {
		return execution, fmt.Errorf("execution created but dispatch failed: %w", err)
	}

	return execution, nil
}

// GetExecution dipakai oleh presentation layer — load WITH steps untuk response lengkap.
func (u *executionUseCaseImpl) GetExecution(ctx context.Context, id uuid.UUID) (*entity.Execution, error) {
	execution, err := u.executionRepo.FindByIDWithSteps(ctx, id)
	if err != nil {
		return nil, err
	}
	if execution == nil {
		return nil, fmt.Errorf("execution not found")
	}
	return execution, nil
}

func (u *executionUseCaseImpl) ListExecutions(ctx context.Context, filter repository.ExecutionFilter, page, pageSize int, sortBy, sortOrder string) ([]entity.Execution, int64, error) {
	return u.executionRepo.FindAll(ctx, filter, page, pageSize, sortBy, sortOrder)
}
