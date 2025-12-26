package app

import (
	"context"
	"fmt"
	"io"

	"github.com/qj0r9j0vc2/alert-bridge/internal/infrastructure/persistence/memory"
	"github.com/qj0r9j0vc2/alert-bridge/internal/infrastructure/persistence/mysql"
	"github.com/qj0r9j0vc2/alert-bridge/internal/infrastructure/persistence/sqlite"
)

func (app *Application) initializeStorage() error {
	var closer io.Closer

	switch app.config.Storage.Type {
	case "mysql":
		repos, db, err := mysql.NewRepositories(&app.config.Storage.MySQL)
		if err != nil {
			return fmt.Errorf("mysql init: %w", err)
		}
		app.alertRepo = repos.Alert
		app.ackEventRepo = repos.AckEvent
		app.silenceRepo = repos.Silence
		closer = db

		app.logger.Get().Info("MySQL storage initialized",
			"host", app.config.Storage.MySQL.Primary.Host,
			"database", app.config.Storage.MySQL.Primary.Database,
		)

	case "sqlite":
		db, err := sqlite.NewDB(app.config.Storage.SQLite.Path)
		if err != nil {
			return fmt.Errorf("sqlite init: %w", err)
		}

		if err := db.Migrate(context.Background()); err != nil {
			db.Close()
			return fmt.Errorf("sqlite migration: %w", err)
		}

		repos := sqlite.NewRepositories(db)
		app.alertRepo = repos.Alert
		app.ackEventRepo = repos.AckEvent
		app.silenceRepo = repos.Silence
		closer = db

		app.logger.Get().Info("SQLite storage initialized",
			"path", app.config.Storage.SQLite.Path,
		)

	case "memory", "":
		app.alertRepo = memory.NewAlertRepository()
		app.ackEventRepo = memory.NewAckEventRepository()
		app.silenceRepo = memory.NewSilenceRepository()

		app.logger.Get().Info("in-memory storage initialized")

	default:
		return fmt.Errorf("unknown storage type: %s", app.config.Storage.Type)
	}

	app.dbCloser = closer
	return nil
}
