package postgres

import (
	"fmt"
	"time"

	"github.com/openspec/api-scheduler-flow-engine/pkg/config"
	"github.com/openspec/api-scheduler-flow-engine/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewConnection creates a new GORM DB connection with connection pooling configured.
func NewConnection(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.DSN()
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// You can configure GORM logger here if needed
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logger.Info("Database connection established")
	return db, nil
}
