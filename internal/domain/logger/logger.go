package logger

// Logger defines the interface for structured logging across the application.
// This interface follows the slog-style logging pattern with key-value pairs.
type Logger interface {
	// Debug logs a debug-level message with optional key-value pairs
	Debug(msg string, keysAndValues ...any)

	// Info logs an info-level message with optional key-value pairs
	Info(msg string, keysAndValues ...any)

	// Warn logs a warning-level message with optional key-value pairs
	Warn(msg string, keysAndValues ...any)

	// Error logs an error-level message with optional key-value pairs
	Error(msg string, keysAndValues ...any)
}
