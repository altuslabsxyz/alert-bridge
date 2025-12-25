package config

import (
	"fmt"
	"log/slog"
	"reflect"
	"sync"

	"github.com/spf13/viper"
)

// ConfigManager manages thread-safe configuration with hot reload support.
type ConfigManager struct {
	mu              sync.RWMutex
	config          *Config
	viper           *viper.Viper
	configPath      string
	logger          *slog.Logger
	onReloadSuccess func(*Config) // Callback after successful reload
}

// NewConfigManager creates a new ConfigManager with the initial configuration.
func NewConfigManager(cfg *Config, v *viper.Viper, configPath string, logger *slog.Logger) *ConfigManager {
	return &ConfigManager{
		config:     cfg,
		viper:      v,
		configPath: configPath,
		logger:     logger,
	}
}

// SetReloadCallback sets a callback function to be called after successful reload.
func (cm *ConfigManager) SetReloadCallback(callback func(*Config)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.onReloadSuccess = callback
}

// Get returns a copy of the current configuration (thread-safe read).
func (cm *ConfigManager) Get() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// TryReload attempts to reload configuration from file.
// Returns error if parsing, validation, or static config changes detected.
// On success, atomically swaps to new configuration.
func (cm *ConfigManager) TryReload() error {
	// Parse new config using the existing Load function
	newCfg, err := Load(cm.configPath)
	if err != nil {
		cm.logger.Error("configuration reload failed",
			"error", err,
			"reason", "parse_error",
			"preserved_config", true,
		)
		return fmt.Errorf("parse failed: %w", err)
	}

	// Check for static config changes
	cm.mu.RLock()
	oldCfg := cm.config
	staticChanges := detectStaticChanges(oldCfg, newCfg)
	cm.mu.RUnlock()

	if len(staticChanges) > 0 {
		// Log warning for static changes
		cm.logger.Warn("configuration change requires restart",
			"changed_keys", staticChanges,
			"reason", getRestartReason(staticChanges[0]),
		)
		return ErrRequiresRestart
	}

	// Extract diff for logging
	cm.mu.RLock()
	diff := extractConfigDiff(cm.config, newCfg)
	cm.mu.RUnlock()

	// Atomic config swap
	cm.mu.Lock()
	cm.config = newCfg
	cm.mu.Unlock()

	// Log successful reload only if there are changes
	if len(diff.ChangedKeys) > 0 {
		cm.logger.Info("configuration reloaded",
			"changed_keys", diff.ChangedKeys,
		)
		for _, key := range diff.ChangedKeys {
			cm.logger.Info("configuration reloaded",
				key, map[string]interface{}{
					"old": diff.OldValues[key],
					"new": diff.NewValues[key],
				},
			)
		}
	}

	// Call reload callback if set
	if cm.onReloadSuccess != nil {
		cm.onReloadSuccess(newCfg)
	}

	return nil
}

// ConfigDiff represents configuration changes.
type ConfigDiff struct {
	ChangedKeys []string
	OldValues   map[string]interface{}
	NewValues   map[string]interface{}
}

// extractConfigDiff compares old and new configs and returns the differences.
func extractConfigDiff(oldCfg, newCfg *Config) ConfigDiff {
	diff := ConfigDiff{
		ChangedKeys: make([]string, 0),
		OldValues:   make(map[string]interface{}),
		NewValues:   make(map[string]interface{}),
	}

	// Compare reloadable fields only
	if oldCfg.Logging.Level != newCfg.Logging.Level {
		diff.ChangedKeys = append(diff.ChangedKeys, "logging.level")
		diff.OldValues["logging.level"] = oldCfg.Logging.Level
		diff.NewValues["logging.level"] = newCfg.Logging.Level
	}

	if oldCfg.Logging.Format != newCfg.Logging.Format {
		diff.ChangedKeys = append(diff.ChangedKeys, "logging.format")
		diff.OldValues["logging.format"] = oldCfg.Logging.Format
		diff.NewValues["logging.format"] = newCfg.Logging.Format
	}

	if oldCfg.Slack.ChannelID != newCfg.Slack.ChannelID {
		diff.ChangedKeys = append(diff.ChangedKeys, "slack.channel_id")
		diff.OldValues["slack.channel_id"] = oldCfg.Slack.ChannelID
		diff.NewValues["slack.channel_id"] = newCfg.Slack.ChannelID
	}

	if oldCfg.Alerting.DeduplicationWindow != newCfg.Alerting.DeduplicationWindow {
		diff.ChangedKeys = append(diff.ChangedKeys, "alerting.deduplication_window")
		diff.OldValues["alerting.deduplication_window"] = oldCfg.Alerting.DeduplicationWindow.String()
		diff.NewValues["alerting.deduplication_window"] = newCfg.Alerting.DeduplicationWindow.String()
	}

	if oldCfg.Alerting.ResendInterval != newCfg.Alerting.ResendInterval {
		diff.ChangedKeys = append(diff.ChangedKeys, "alerting.resend_interval")
		diff.OldValues["alerting.resend_interval"] = oldCfg.Alerting.ResendInterval.String()
		diff.NewValues["alerting.resend_interval"] = newCfg.Alerting.ResendInterval.String()
	}

	return diff
}

// detectStaticChanges checks if any static (restart-required) config has changed.
func detectStaticChanges(oldCfg, newCfg *Config) []string {
	changes := make([]string, 0)

	// Server config (static)
	if oldCfg.Server.Port != newCfg.Server.Port {
		changes = append(changes, "server.port")
	}

	// Storage type (static)
	if oldCfg.Storage.Type != newCfg.Storage.Type {
		changes = append(changes, "storage.type")
	}

	// SQLite path (static)
	if oldCfg.Storage.SQLite.Path != newCfg.Storage.SQLite.Path {
		changes = append(changes, "storage.sqlite.path")
	}

	// MySQL config (static)
	if !reflect.DeepEqual(oldCfg.Storage.MySQL, newCfg.Storage.MySQL) {
		changes = append(changes, "storage.mysql")
	}

	return changes
}

// ErrRequiresRestart is returned when static configuration changes are detected.
var ErrRequiresRestart = fmt.Errorf("configuration change requires application restart")
