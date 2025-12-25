package config

import (
	"log/slog"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Watcher manages configuration file watching with hot reload.
type Watcher struct {
	viper          *viper.Viper
	configManager  *ConfigManager
	logger         *slog.Logger
	debounceTimer  *time.Timer
	debounceMu     sync.Mutex
	debouncePeriod time.Duration
}

// NewWatcher creates a new configuration file watcher.
func NewWatcher(v *viper.Viper, cm *ConfigManager, logger *slog.Logger) *Watcher {
	return &Watcher{
		viper:          v,
		configManager:  cm,
		logger:         logger,
		debouncePeriod: 100 * time.Millisecond,
	}
}

// Start begins watching the configuration file for changes.
func (w *Watcher) Start() {
	w.viper.WatchConfig()
	w.viper.OnConfigChange(w.onConfigChange)
	w.logger.Info("config watcher started",
		"watch_path", w.viper.ConfigFileUsed(),
	)
}

// onConfigChange handles configuration file change events with debouncing.
func (w *Watcher) onConfigChange(e fsnotify.Event) {
	w.debounceMu.Lock()
	defer w.debounceMu.Unlock()

	// Stop existing timer if any
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}

	// Check if file was deleted
	if e.Op&fsnotify.Remove == fsnotify.Remove {
		w.logger.Error("config file removed",
			"file", e.Name,
			"preserved_config", true,
		)
		return
	}

	// Start new debounce timer
	w.debounceTimer = time.AfterFunc(w.debouncePeriod, func() {
		if err := w.configManager.TryReload(); err != nil {
			if err == ErrRequiresRestart {
				// Already logged in TryReload with WARNING level
				return
			}
			// Error already logged in TryReload with ERROR level
		}
	})
}
