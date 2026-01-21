package app

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/altuslabsxyz/alert-bridge/internal/infrastructure/config"
)

func (app *Application) loadConfig(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	app.config = cfg
	return nil
}

func (app *Application) setupConfigManager(configPath string) error {
	v := viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("initializing viper: %w", err)
	}

	app.configManager = config.NewConfigManager(app.config, v, configPath, app.logger.Get())

	// Setup reload callback for logger
	app.configManager.SetReloadCallback(func(newCfg *config.Config) {
		newLogger := createLogger(newCfg.Logging.Level, newCfg.Logging.Format)
		app.logger.Set(newLogger)
		app.logger.Get().Info("logger reloaded",
			"level", newCfg.Logging.Level,
			"format", newCfg.Logging.Format,
		)
	})

	return nil
}
