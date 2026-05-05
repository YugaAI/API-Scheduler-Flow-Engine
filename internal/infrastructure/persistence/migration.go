package persistence

import (
	"fmt"

	"github.com/openspec/api-scheduler-flow-engine/internal/domain/entity"
	"github.com/openspec/api-scheduler-flow-engine/pkg/logger"
	"gorm.io/gorm"
)

// RunMigrations executes auto-migration for all domain entities.
func RunMigrations(db *gorm.DB) error {
	logger.Info("Running database migrations")

	// Ensure uuid-ossp extension is enabled for gen_random_uuid() if using older postgres
	// Postgres 13+ supports gen_random_uuid() natively.
	err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error
	if err != nil {
		logger.Warn("Failed to create uuid-ossp extension (might not be needed for PG13+)", "error", err)
	}

	err = db.AutoMigrate(
		&entity.Flow{},
		&entity.Step{},
		&entity.Execution{},
		&entity.ExecutionStep{},
		&entity.Schedule{},
	)
	if err != nil {
		return fmt.Errorf("auto-migrate failed: %w", err)
	}

	logger.Info("Database migrations completed successfully")
	return nil
}
