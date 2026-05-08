package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// closer holds the log file handle so it can be closed on shutdown.
var closer io.Closer

// Init initializes the global logger with a JSON handler.
// Output is written to both stdout and a log file (if LOG_FILE env is set or logDir is non-empty).
// Allowed levels: "debug", "info", "warn", "error". Defaults to "info" on unknown level.
func Init(levelStr string) {
	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	output := buildOutput()

	opts := &slog.HandlerOptions{Level: level}
	handler := slog.NewJSONHandler(output, opts)
	slog.SetDefault(slog.New(handler))
}

// Close flushes and closes the underlying log file (if any).
// Call this in main() via defer logger.Close() after logger.Init().
func Close() {
	if closer != nil {
		_ = closer.Close()
	}
}

// buildOutput returns an io.Writer that writes to stdout,
// and additionally to a log file when LOG_FILE env var is set.
func buildOutput() io.Writer {
	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		return os.Stdout
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "logger: cannot create log directory: %v\n", err)
		return os.Stdout
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: cannot open log file %q: %v — falling back to stdout\n", logFile, err)
		return os.Stdout
	}

	closer = f
	return io.MultiWriter(os.Stdout, f)
}

// Info logs an informational message.
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}
