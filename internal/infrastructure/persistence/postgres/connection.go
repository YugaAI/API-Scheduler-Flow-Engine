package postgres

import (
	"fmt"
	"time"

	"github.com/openspec/api-scheduler-flow-engine/pkg/config"
	"github.com/openspec/api-scheduler-flow-engine/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// NewConnection creates a new GORM DB connection with connection pooling
// dan GORM slow query logger yang konsisten dengan app logger.
func NewConnection(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.DSN()

	// GORM logger: log slow query >= 200ms ke stdout
	// Level Warn agar query normal tidak noise di log
	gormLog := gormlogger.Default.LogMode(gormlogger.Warn)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLog,
		// PrepareStmt: true — cache prepared statements, tingkatkan throughput query berulang
		PrepareStmt: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Connection pool tuning:
	// MaxIdleConns  — koneksi idle yang tetap hidup (hindari reconnect overhead)
	// MaxOpenConns  — batas total koneksi ke PostgreSQL
	// ConnMaxLifetime — rotasi koneksi untuk hindari stale connection
	// ConnMaxIdleTime — tutup koneksi idle > 10 menit (efisiensi resource)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	logger.Info("Database connection established")
	return db, nil
}
