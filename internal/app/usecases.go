package app

import (
	"log/slog"

	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/service"
	"github.com/qj0r9j0vc2/alert-bridge/internal/usecase/ack"
	"github.com/qj0r9j0vc2/alert-bridge/internal/usecase/alert"
)

// UseCases holds all business logic use cases
type UseCases struct {
	ProcessAlert      *alert.ProcessAlertUseCase
	SyncAck           *ack.SyncAckUseCase
	SubscriberMatcher *service.SubscriberMatcher
}

func (app *Application) initializeUseCases() error {
	logger := &slogAdapter{logger: app.logger.Get()}

	processAlertUseCase := alert.NewProcessAlertUseCase(
		app.alertRepo,
		app.silenceRepo,
		app.clients.Notifiers,
		logger,
		app.telemetry.Metrics,
	)

	// Initialize subscriber matcher if subscribers are configured
	var subscriberMatcher *service.SubscriberMatcher
	if len(app.config.Subscribers) > 0 {
		subscriberMatcher = service.NewSubscriberMatcher(app.config.GetEnabledSubscribers())
		processAlertUseCase.SetSubscriberMatcher(subscriberMatcher)

		app.logger.Get().Info("subscriber matching enabled",
			"subscriberCount", len(app.config.GetEnabledSubscribers()),
		)

		// Set up subscriber-aware notifiers
		if app.clients.Slack != nil {
			slackAdapter := NewSlackSubscriberNotifierAdapter(app.clients.Slack)
			processAlertUseCase.SetSlackSubscriberNotifier(slackAdapter)
		}

		if app.clients.PagerDuty != nil {
			pagerDutyAdapter := NewPagerDutySubscriberNotifierAdapter(app.clients.PagerDuty)
			processAlertUseCase.SetPagerDutySubscriberNotifier(pagerDutyAdapter)
		}
	}

	app.useCases = &UseCases{
		ProcessAlert:      processAlertUseCase,
		SyncAck: ack.NewSyncAckUseCase(
			app.alertRepo,
			app.ackEventRepo,
			app.txManager,
			app.clients.Syncers,
			logger,
			app.telemetry.Metrics,
		),
		SubscriberMatcher: subscriberMatcher,
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
