package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Server
	ServerPort string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis
	RedisURL string

	// JWT
	JWTSecret string

	// Worker Pool
	WorkerPoolSize int

	// Scheduler
	Timezone string

	// Execution
	ExecutionTimeoutSeconds int
	DefaultRetryCount       int
	DefaultRetryDelay       int
	FailurePolicy           string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		ServerPort:              getEnv("SERVER_PORT", "8080"),
		DBHost:                  getEnv("DB_HOST", "localhost"),
		DBPort:                  getEnv("DB_PORT", "5432"),
		DBUser:                  getEnv("DB_USER", "postgres"),
		DBPassword:              getEnv("DB_PASSWORD", "postgres"),
		DBName:                  getEnv("DB_NAME", "flow_engine"),
		DBSSLMode:               getEnv("DB_SSL_MODE", "disable"),
		RedisURL:                getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:               getEnv("JWT_SECRET", ""),
		Timezone:                getEnv("TIMEZONE", "Asia/Jakarta"),
		FailurePolicy:           getEnv("FAILURE_POLICY", "stop"),
	}

	var err error

	cfg.WorkerPoolSize, err = getEnvInt("WORKER_POOL_SIZE", 10)
	if err != nil {
		return nil, fmt.Errorf("invalid WORKER_POOL_SIZE: %w", err)
	}

	cfg.ExecutionTimeoutSeconds, err = getEnvInt("EXECUTION_TIMEOUT_SECONDS", 300)
	if err != nil {
		return nil, fmt.Errorf("invalid EXECUTION_TIMEOUT_SECONDS: %w", err)
	}

	cfg.DefaultRetryCount, err = getEnvInt("DEFAULT_RETRY_COUNT", 3)
	if err != nil {
		return nil, fmt.Errorf("invalid DEFAULT_RETRY_COUNT: %w", err)
	}

	cfg.DefaultRetryDelay, err = getEnvInt("DEFAULT_RETRY_DELAY", 5)
	if err != nil {
		return nil, fmt.Errorf("invalid DEFAULT_RETRY_DELAY: %w", err)
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	return cfg, nil
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) (int, error) {
	valStr, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal, nil
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, fmt.Errorf("cannot parse %s=%q as int: %w", key, valStr, err)
	}
	return val, nil
}
