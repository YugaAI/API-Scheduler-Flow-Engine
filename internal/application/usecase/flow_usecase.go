package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/application/service"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/repository"
)

type FlowUseCase interface {
	CreateFlow(ctx context.Context, flow *entity.Flow) error
	GetFlow(ctx context.Context, id uuid.UUID) (*entity.Flow, error)
	ListFlows(ctx context.Context, page, pageSize int) ([]entity.Flow, int64, error)
	UpdateFlow(ctx context.Context, flow *entity.Flow) error
	DeleteFlow(ctx context.Context, id uuid.UUID) error
}

type flowUseCaseImpl struct {
	flowRepo       repository.FlowRepository
	actionRegistry *service.ActionRegistry
}

func NewFlowUseCase(flowRepo repository.FlowRepository, actionRegistry *service.ActionRegistry) FlowUseCase {
	return &flowUseCaseImpl{
		flowRepo:       flowRepo,
		actionRegistry: actionRegistry,
	}
}

func (u *flowUseCaseImpl) CreateFlow(ctx context.Context, flow *entity.Flow) error {
	if len(flow.Steps) == 0 {
		return fmt.Errorf("at least one step is required")
	}

	if err := u.validateSteps(flow.Steps); err != nil {
		return err
	}

	return u.flowRepo.Create(ctx, flow)
}

func (u *flowUseCaseImpl) GetFlow(ctx context.Context, id uuid.UUID) (*entity.Flow, error) {
	flow, err := u.flowRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if flow == nil {
		return nil, fmt.Errorf("flow not found")
	}
	return flow, nil
}

func (u *flowUseCaseImpl) ListFlows(ctx context.Context, page, pageSize int) ([]entity.Flow, int64, error) {
	return u.flowRepo.FindAll(ctx, page, pageSize)
}

func (u *flowUseCaseImpl) UpdateFlow(ctx context.Context, flow *entity.Flow) error {
	existing, err := u.flowRepo.FindByID(ctx, flow.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("flow not found")
	}

	if len(flow.Steps) > 0 {
		if err := u.validateSteps(flow.Steps); err != nil {
			return err
		}
	}

	return u.flowRepo.Update(ctx, flow)
}

func (u *flowUseCaseImpl) DeleteFlow(ctx context.Context, id uuid.UUID) error {
	return u.flowRepo.Delete(ctx, id)
}

func (u *flowUseCaseImpl) validateSteps(steps []entity.Step) error {
	orderMap := make(map[int]bool)
	for _, step := range steps {
		if !u.actionRegistry.Validate(step.Action) {
			return fmt.Errorf("unknown action: %s", step.Action)
		}
		if orderMap[step.Order] {
			return fmt.Errorf("step orders must be unique")
		}
		orderMap[step.Order] = true
	}
	return nil
}
