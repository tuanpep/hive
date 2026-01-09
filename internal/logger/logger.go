// Package logger provides structured logging for the orchestrator.
package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/tuanbt/hive/internal/config"
)

// NewSystemLogger creates the main orchestrator logger.
func NewSystemLogger(cfg *config.Config) (*slog.Logger, error) {
	level := ParseLevel(cfg.LogLevel)

	// Ensure log directory exists
	if err := os.MkdirAll(cfg.LogDirectory, 0755); err != nil {
		return nil, err
	}

	// Create log file
	logPath := filepath.Join(cfg.LogDirectory, "orchestrator.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// Multi-writer: file + stdout
	multiWriter := io.MultiWriter(os.Stdout, file)

	// JSON handler for structured logs
	handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler), nil
}

// NewEmbeddedLogger creates a logger that ONLY writes to file (for TUI embedding).
func NewEmbeddedLogger(cfg *config.Config) (*slog.Logger, error) {
	level := ParseLevel(cfg.LogLevel)

	// Ensure log directory exists
	if err := os.MkdirAll(cfg.LogDirectory, 0755); err != nil {
		return nil, err
	}

	// Create log file
	logPath := filepath.Join(cfg.LogDirectory, "orchestrator.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// File ONLY, no stdout
	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler), nil
}

// NewTaskLogger creates a logger for a specific task.
// Returns the logger and a cleanup function to close the file.
func NewTaskLogger(cfg *config.Config, taskID string) (*slog.Logger, func(), error) {
	level := ParseLevel(cfg.LogLevel)

	// Ensure log directory exists
	if err := os.MkdirAll(cfg.LogDirectory, 0755); err != nil {
		return nil, nil, err
	}

	// Create task log file
	logPath := filepath.Join(cfg.LogDirectory, taskID+".log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler).With("task_id", taskID)
	cleanup := func() { file.Close() }

	return logger, cleanup, nil
}

// NewConsoleLogger creates a simple console-only logger.
func NewConsoleLogger(cfg *config.Config) *slog.Logger {
	level := ParseLevel(cfg.LogLevel)

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}

// ParseLevel converts a string log level to slog.Level.
func ParseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
