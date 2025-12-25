package config

import (
	"fmt"
	"time"
)

// reloadableKeys defines the whitelist of configuration keys that can be hot-reloaded.
var reloadableKeys = map[string]bool{
	"logging.level":                  true,
	"logging.format":                 true,
	"slack.channel_id":               true,
	"alerting.deduplication_window":  true,
	"alerting.resend_interval":       true,
}

// staticKeys defines configuration keys that require application restart.
var staticKeys = map[string]string{
	"server.port":         "HTTP listener restart required",
	"storage.type":        "Storage backend initialization required",
	"storage.sqlite.path": "Database connection recreation required",
	"storage.mysql":       "Database connection pool recreation required",
}

// IsReloadable returns true if the given config key can be hot-reloaded.
func IsReloadable(key string) bool {
	return reloadableKeys[key]
}

// GetRestartReason returns the reason why a static config key requires restart.
func getRestartReason(key string) string {
	if reason, ok := staticKeys[key]; ok {
		return reason
	}
	return "unknown configuration requires restart"
}

// ValidateLogLevel checks if the log level is valid.
func ValidateLogLevel(level string) error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", level)
	}
	return nil
}

// ValidateLogFormat checks if the log format is valid.
func ValidateLogFormat(format string) error {
	validFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validFormats[format] {
		return fmt.Errorf("invalid log format: %s (must be json or text)", format)
	}
	return nil
}

// ValidateNonEmpty checks if a string is non-empty.
func ValidateNonEmpty(value string, fieldName string) error {
	if value == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	return nil
}

// ValidateDuration checks if a duration is greater than zero.
func ValidateDuration(duration time.Duration, fieldName string) error {
	if duration <= 0 {
		return fmt.Errorf("%s must be greater than 0", fieldName)
	}
	return nil
}
