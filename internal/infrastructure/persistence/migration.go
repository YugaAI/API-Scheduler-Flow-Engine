package persistence

import (
	"fmt"

	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/pkg/logger"
	"gorm.io/gorm"
)

// RunMigrations executes auto-migration for all domain entities.
// Index eksplisit ditambahkan untuk query yang sering dipakai di execution path.
func RunMigrations(db *gorm.DB) error {
	logger.Info("Running database migrations")

	// uuid-ossp untuk PG < 13 compatibility — tidak fatal jika gagal
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		logger.Warn("Failed to create uuid-ossp extension (not needed for PG13+)", "error", err)
	}

	if err := db.AutoMigrate(
		&entity.Flow{},
		&entity.Step{},
		&entity.Execution{},
		&entity.ExecutionStep{},
		&entity.Schedule{},
	); err != nil {
		return fmt.Errorf("auto-migrate failed: %w", err)
	}

	// Index eksplisit untuk query yang sering dipakai di hot path:
	//
	// 1. executions.flow_id — sering difilter di ListExecutions
	// 2. executions.status  — sering difilter di ListExecutions
	// 3. executions.started_at — default sort column di FindAll
	// 4. execution_steps.execution_id + step_order — dipakai di preload & UpdateStep
	//
	// AutoMigrate sudah menangani FK index, tapi index komposit dan sort-index perlu manual.
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_executions_flow_id ON executions(flow_id)`,
		`CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_executions_started_at ON executions(started_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_executions_created_at ON executions(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_execution_steps_execution_id ON execution_steps(execution_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_execution_steps_exec_order ON execution_steps(execution_id, step_order)`,
		`CREATE INDEX IF NOT EXISTS idx_steps_flow_id ON steps(flow_id)`,
		`CREATE INDEX IF NOT EXISTS idx_schedules_flow_id ON schedules(flow_id)`,
		`CREATE INDEX IF NOT EXISTS idx_schedules_enabled ON schedules(enabled) WHERE enabled = true`,
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			// Non-fatal — index bisa sudah ada atau gagal karena permission
			logger.Warn("Failed to create index (non-fatal)", "sql", idx, "error", err)
		}
	}

	logger.Info("Database migrations completed successfully")
	return nil
}
