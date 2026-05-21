package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/internal/domain/repository"
	"gorm.io/gorm"
)

// allowedSortBy adalah whitelist kolom yang valid untuk sorting executions.
// Digunakan untuk mencegah SQL injection pada query FindAll.
var allowedSortBy = map[string]bool{
	"started_at":  true,
	"finished_at": true,
	"created_at":  true,
	"status":      true,
}

// allowedSortOrder adalah whitelist direction sorting yang valid.
var allowedSortOrder = map[string]bool{
	"asc":  true,
	"desc": true,
}

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

// FindByID memuat execution TANPA steps — dipakai di executor path untuk metadata saja.
// Menghindari JOIN tidak perlu yang menyebabkan SLOW SQL 215ms.
func (r *executionRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entity.Execution, error) {
	var execution entity.Execution
	err := r.db.WithContext(ctx).
		Select("id", "flow_id", "status", "trigger_type", "started_at", "finished_at", "created_at", "updated_at").
		First(&execution, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &execution, nil
}

// FindByIDWithSteps memuat execution DENGAN steps preloaded — dipakai di presentation layer.
func (r *executionRepositoryImpl) FindByIDWithSteps(ctx context.Context, id uuid.UUID) (*entity.Execution, error) {
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

	if filter.FlowID != nil {
		query = query.Where("flow_id = ?", *filter.FlowID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Whitelist validation — cegah SQL injection
	if !allowedSortBy[sortBy] {
		sortBy = "started_at"
	}
	if !allowedSortOrder[sortOrder] {
		sortOrder = "desc"
	}
	orderClause := fmt.Sprintf("%s %s", sortBy, sortOrder)

	offset := (page - 1) * pageSize
	err := query.Order(orderClause).Offset(offset).Limit(pageSize).Find(&executions).Error
	if err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}

// Update hanya meng-update field status, started_at, finished_at.
// Menghindari full Save() yang overwrite semua field dan berisiko race condition.
func (r *executionRepositoryImpl) Update(ctx context.Context, execution *entity.Execution) error {
	return r.db.WithContext(ctx).
		Model(execution).
		Select("status", "started_at", "finished_at").
		Updates(map[string]interface{}{
			"status":      execution.Status,
			"started_at":  execution.StartedAt,
			"finished_at": execution.FinishedAt,
		}).Error
}

// CreateStep insert row baru untuk execution step.
// Harus dipanggil pertama kali sebelum UpdateStep agar ID terisi oleh GORM.
func (r *executionRepositoryImpl) CreateStep(ctx context.Context, step *entity.ExecutionStep) error {
	return r.db.WithContext(ctx).Create(step).Error
}

// UpdateStep hanya meng-update field yang relevan — menghindari full Save().
// Aman dari race condition karena tidak overwrite field yang tidak berubah.
func (r *executionRepositoryImpl) UpdateStep(ctx context.Context, step *entity.ExecutionStep) error {
	return r.db.WithContext(ctx).
		Model(step).
		Select("status", "log", "retry_attempts", "started_at", "finished_at").
		Updates(map[string]interface{}{
			"status":         step.Status,
			"log":            step.Log,
			"retry_attempts": step.RetryAttempts,
			"started_at":     step.StartedAt,
			"finished_at":    step.FinishedAt,
		}).Error
}
