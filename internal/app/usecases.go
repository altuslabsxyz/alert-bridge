package app

import (
	"log/slog"

	"github.com/qj0r9j0vc2/alert-bridge/internal/usecase/ack"
	"github.com/qj0r9j0vc2/alert-bridge/internal/usecase/alert"
)

// UseCases holds all business logic use cases
type UseCases struct {
	ProcessAlert *alert.ProcessAlertUseCase
	SyncAck      *ack.SyncAckUseCase
}

func (app *Application) initializeUseCases() error {
	logger := &slogAdapter{logger: app.logger.Get()}

	app.useCases = &UseCases{
		ProcessAlert: alert.NewProcessAlertUseCase(
			app.alertRepo,
			app.silenceRepo,
			app.clients.Notifiers,
			logger,
			app.telemetry.Metrics,
		),
		SyncAck: ack.NewSyncAckUseCase(
			app.alertRepo,
			app.ackEventRepo,
			app.txManager,
			app.clients.Syncers,
			logger,
			app.telemetry.Metrics,
		),
	}

	return nil
}

// slogAdapter adapts slog.Logger to usecase Logger interface
type slogAdapter struct {
	logger *slog.Logger
}

func (a *slogAdapter) Debug(msg string, keysAndValues ...any) {
	a.logger.Debug(msg, keysAndValues...)
}

func (a *slogAdapter) Info(msg string, keysAndValues ...any) {
	a.logger.Info(msg, keysAndValues...)
}

func (a *slogAdapter) Warn(msg string, keysAndValues ...any) {
	a.logger.Warn(msg, keysAndValues...)
}

func (a *slogAdapter) Error(msg string, keysAndValues ...any) {
	a.logger.Error(msg, keysAndValues...)
}
