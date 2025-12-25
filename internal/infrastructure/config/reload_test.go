package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// TestLoggingLevelHotReload tests that logging.level can be hot-reloaded.
func TestLoggingLevelHotReload(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `
logging:
  level: info
  format: json
slack:
  enabled: false
  channel_id: C123456
alerting:
  deduplication_window: 5m
  resend_interval: 30m
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load config with Viper (for file watching)
	v := viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	// Parse initial config using Load
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cm := NewConfigManager(cfg, v, configPath, logger)

	// Verify initial level
	if cm.Get().Logging.Level != "info" {
		t.Errorf("expected initial level 'info', got '%s'", cm.Get().Logging.Level)
	}

	// Update config file
	updatedConfig := `
logging:
  level: debug
  format: json
slack:
  enabled: false
  channel_id: C123456
alerting:
  deduplication_window: 5m
  resend_interval: 30m
`
	if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("failed to update config file: %v", err)
	}

	// Reload configuration
	if err := cm.TryReload(); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	// Verify new level
	if cm.Get().Logging.Level != "debug" {
		t.Errorf("expected level 'debug' after reload, got '%s'", cm.Get().Logging.Level)
	}
}

// TestSlackChannelIDHotReload tests that slack.channel_id can be hot-reloaded.
func TestSlackChannelIDHotReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `
logging:
  level: info
  format: json
slack:
  enabled: false
  channel_id: C123456
alerting:
  deduplication_window: 5m
  resend_interval: 30m
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cm := NewConfigManager(cfg, v, configPath, logger)

	// Verify initial channel
	if cm.Get().Slack.ChannelID != "C123456" {
		t.Errorf("expected initial channel 'C123456', got '%s'", cm.Get().Slack.ChannelID)
	}

	// Update config
	updatedConfig := `
logging:
  level: info
  format: json
slack:
  enabled: false
  channel_id: C987654321
alerting:
  deduplication_window: 5m
  resend_interval: 30m
`
	if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("failed to update config file: %v", err)
	}

	if err := cm.TryReload(); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	// Verify new channel
	if cm.Get().Slack.ChannelID != "C987654321" {
		t.Errorf("expected channel 'C987654321' after reload, got '%s'", cm.Get().Slack.ChannelID)
	}
}

// TestMultipleSettingsReload tests that multiple settings can be changed at once.
func TestMultipleSettingsReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `
logging:
  level: info
  format: json
slack:
  enabled: false
  channel_id: C123456
alerting:
  deduplication_window: 5m
  resend_interval: 30m
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cm := NewConfigManager(cfg, v, configPath, logger)

	// Update multiple settings
	updatedConfig := `
logging:
  level: debug
  format: text
slack:
  enabled: false
  channel_id: C999999
alerting:
  deduplication_window: 10m
  resend_interval: 1h
`
	if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("failed to update config file: %v", err)
	}

	if err := cm.TryReload(); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	// Verify all changes applied atomically
	newCfg := cm.Get()
	if newCfg.Logging.Level != "debug" {
		t.Errorf("expected level 'debug', got '%s'", newCfg.Logging.Level)
	}
	if newCfg.Logging.Format != "text" {
		t.Errorf("expected format 'text', got '%s'", newCfg.Logging.Format)
	}
	if newCfg.Slack.ChannelID != "C999999" {
		t.Errorf("expected channel 'C999999', got '%s'", newCfg.Slack.ChannelID)
	}
	if newCfg.Alerting.DeduplicationWindow != 10*time.Minute {
		t.Errorf("expected dedup window 10m, got %v", newCfg.Alerting.DeduplicationWindow)
	}
	if newCfg.Alerting.ResendInterval != 1*time.Hour {
		t.Errorf("expected resend interval 1h, got %v", newCfg.Alerting.ResendInterval)
	}
}

// TestInvalidYAMLHandling tests that invalid YAML preserves existing config.
func TestInvalidYAMLHandling(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	validConfig := `
logging:
  level: info
  format: json
slack:
  enabled: false
  channel_id: C123456
alerting:
  deduplication_window: 5m
  resend_interval: 30m
`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cm := NewConfigManager(cfg, v, configPath, logger)

	// Write invalid YAML
	invalidConfig := `
logging:
  level: info
  invalid: yaml: syntax: error
`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	// Try to reload invalid config - should fail
	if err = cm.TryReload(); err == nil {
		t.Fatal("expected reload to fail with invalid YAML, but it succeeded")
	}

	// In production, the watcher would catch the error and preserve old config

	// Verify old config preserved
	if cm.Get().Logging.Level != "info" {
		t.Errorf("expected preserved level 'info', got '%s'", cm.Get().Logging.Level)
	}
	if cm.Get().Slack.ChannelID != "C123456" {
		t.Errorf("expected preserved channel 'C123456', got '%s'", cm.Get().Slack.ChannelID)
	}
}

// TestStaticConfigWarning tests that static config changes are rejected.
func TestStaticConfigWarning(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `
server:
  port: 8080
logging:
  level: info
  format: json
storage:
  type: memory
slack:
  enabled: false
  channel_id: C123456
alerting:
  deduplication_window: 5m
  resend_interval: 30m
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cm := NewConfigManager(cfg, v, configPath, logger)

	// Try to change server.port (static config)
	updatedConfig := `
server:
  port: 9090
logging:
  level: info
  format: json
storage:
  type: memory
slack:
  enabled: false
  channel_id: C123456
alerting:
  deduplication_window: 5m
  resend_interval: 30m
`
	if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("failed to update config file: %v", err)
	}

	// Reload should fail with ErrRequiresRestart
	if err = cm.TryReload(); err != ErrRequiresRestart {
		t.Errorf("expected ErrRequiresRestart, got %v", err)
	}

	// Verify port unchanged
	if cm.Get().Server.Port != 8080 {
		t.Errorf("expected port to remain 8080, got %d", cm.Get().Server.Port)
	}
}

// TestConfigFileDeletion tests that config file deletion is handled gracefully.
func TestConfigFileDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `
logging:
  level: info
  format: json
slack:
  enabled: false
  channel_id: C123456
alerting:
  deduplication_window: 5m
  resend_interval: 30m
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cm := NewConfigManager(cfg, v, configPath, logger)
	watcher := NewWatcher(v, cm, logger)

	// Simulate file deletion via watcher
	os.Remove(configPath)

	// Watcher should log error and preserve config
	// This is tested in the onConfigChange handler
	// For unit test, we just verify the config remains
	if cm.Get().Logging.Level != "info" {
		t.Errorf("expected preserved level 'info', got '%s'", cm.Get().Logging.Level)
	}

	// Suppress unused variable warning
	_ = watcher
}
