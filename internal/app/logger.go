package app

import (
	"log/slog"
	"os"
	"sync/atomic"
)

// AtomicLogger provides thread-safe logger access for hot reload
type AtomicLogger struct {
	value atomic.Value
}

// NewAtomicLogger creates a new atomic logger wrapper
func NewAtomicLogger(logger *slog.Logger) *AtomicLogger {
	al := &AtomicLogger{}
	al.value.Store(logger)
	return al
}

// Get returns the current logger instance
func (al *AtomicLogger) Get() *slog.Logger {
	return al.value.Load().(*slog.Logger)
}

// Set updates the logger instance (thread-safe)
func (al *AtomicLogger) Set(logger *slog.Logger) {
	al.value.Store(logger)
}

// setupLogger creates the initial logger
func (app *Application) setupLogger() error {
	logger := createLogger(app.config.Logging.Level, app.config.Logging.Format)
	app.logger = NewAtomicLogger(logger)
	return nil
}

func createLogger(level, format string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: logLevel}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
