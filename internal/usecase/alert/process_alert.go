package alert

import (
	"context"
	"fmt"
	"time"

	"github.com/qj0r9j0vc2/alert-bridge/internal/adapter/dto"
	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/entity"
	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/repository"
	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/service"
	"github.com/qj0r9j0vc2/alert-bridge/internal/infrastructure/observability"
)

// ProcessAlertUseCase handles incoming alerts from Alertmanager.
type ProcessAlertUseCase struct {
	alertRepo   repository.AlertRepository
	silenceRepo repository.SilenceRepository
	notifiers   []Notifier
	logger      Logger
	metrics     *observability.Metrics

	// Subscriber matching support (optional)
	subscriberMatcher *service.SubscriberMatcher
	slackNotifier     SlackSubscriberNotifier
	pagerDutyNotifier PagerDutySubscriberNotifier
}

// NewProcessAlertUseCase creates a new ProcessAlertUseCase with dependencies.
func NewProcessAlertUseCase(
	alertRepo repository.AlertRepository,
	silenceRepo repository.SilenceRepository,
	notifiers []Notifier,
	logger Logger,
	metrics *observability.Metrics,
) *ProcessAlertUseCase {
	return &ProcessAlertUseCase{
		alertRepo:   alertRepo,
		silenceRepo: silenceRepo,
		notifiers:   notifiers,
		logger:      logger,
		metrics:     metrics,
	}
}

// SetSubscriberMatcher sets the subscriber matcher for label-based subscriber matching.
// This enables @mentions in Slack and sequential PagerDuty escalation.
func (uc *ProcessAlertUseCase) SetSubscriberMatcher(matcher *service.SubscriberMatcher) {
	uc.subscriberMatcher = matcher
}

// SetSlackSubscriberNotifier sets the Slack notifier that supports subscriber mentions.
func (uc *ProcessAlertUseCase) SetSlackSubscriberNotifier(notifier SlackSubscriberNotifier) {
	uc.slackNotifier = notifier
}

// SetPagerDutySubscriberNotifier sets the PagerDuty notifier that supports subscriber escalation.
func (uc *ProcessAlertUseCase) SetPagerDutySubscriberNotifier(notifier PagerDutySubscriberNotifier) {
	uc.pagerDutyNotifier = notifier
}

// Execute processes an incoming alert.
func (uc *ProcessAlertUseCase) Execute(ctx context.Context, input dto.ProcessAlertInput) (*dto.ProcessAlertOutput, error) {
	start := time.Now()
	success := false

	defer func() {
		duration := time.Since(start)
		if uc.metrics != nil {
			uc.metrics.RecordAlertProcessed(
				ctx,
				input.Name,
				string(input.Severity),
				input.Status,
				duration,
				success,
			)
		}
	}()

	output := &dto.ProcessAlertOutput{}

	// 1. Check if alert exists (by fingerprint)
	existing, err := uc.alertRepo.FindByFingerprint(ctx, input.Fingerprint)
	if err != nil {
		return nil, fmt.Errorf("finding alert by fingerprint: %w", err)
	}

	var alert *entity.Alert

	// 2. Handle based on status
	if input.Status == "resolved" {
		// Find the firing alert to resolve
		alert = uc.findFiringAlert(existing)
		if alert == nil {
			// No firing alert to resolve, skip
			uc.logger.Debug("no firing alert found to resolve",
				"fingerprint", input.Fingerprint,
			)
			success = true
			return output, nil
		}

		// Resolve the alert
		alert.Resolve(time.Now().UTC())
		if err := uc.alertRepo.Update(ctx, alert); err != nil {
			return nil, fmt.Errorf("updating resolved alert: %w", err)
		}

		output.AlertID = alert.ID
		output.IsNew = false

		// Update notifications to show resolved state
		uc.updateNotifications(ctx, alert, output)

		success = true
		return output, nil
	}

	// Status is "firing"
	// 3. Check if we already have a firing alert for this fingerprint
	alert = uc.findFiringAlert(existing)
	if alert != nil {
		// Already have a firing alert, skip (deduplication)
		uc.logger.Debug("alert already firing, skipping",
			"alertID", alert.ID,
			"fingerprint", input.Fingerprint,
		)
		output.AlertID = alert.ID
		output.IsNew = false
		success = true
		return output, nil
	}

	// 4. Create new alert
	alert = entity.NewAlert(
		input.Fingerprint,
		input.Name,
		input.Instance,
		input.Target,
		input.Summary,
		input.Severity,
	)
	alert.Description = input.Description
	alert.FiredAt = input.FiredAt

	// Copy labels and annotations
	for k, v := range input.Labels {
		alert.AddLabel(k, v)
	}
	for k, v := range input.Annotations {
		alert.AddAnnotation(k, v)
	}

	// 5. Check if alert is silenced
	silences, err := uc.silenceRepo.FindMatchingAlert(ctx, alert)
	if err != nil {
		uc.logger.Warn("failed to check silences",
			"error", err,
			"alertID", alert.ID,
		)
	}

	if len(silences) > 0 {
		uc.logger.Info("alert is silenced",
			"alertID", alert.ID,
			"silenceID", silences[0].ID,
			"silenceEndAt", silences[0].EndAt,
		)
		output.IsSilenced = true

		// Still save the alert for tracking, but don't notify
		if err := uc.alertRepo.Save(ctx, alert); err != nil {
			return nil, fmt.Errorf("saving silenced alert: %w", err)
		}

		output.AlertID = alert.ID
		output.IsNew = true
		success = true
		return output, nil
	}

	// 6. Save alert
	if err := uc.alertRepo.Save(ctx, alert); err != nil {
		return nil, fmt.Errorf("saving alert: %w", err)
	}

	output.AlertID = alert.ID
	output.IsNew = true

	// 7. Send notifications
	uc.sendNotifications(ctx, alert, output)

	success = true
	return output, nil
}

// findFiringAlert finds a firing (non-resolved) alert from the list.
func (uc *ProcessAlertUseCase) findFiringAlert(alerts []*entity.Alert) *entity.Alert {
	for _, alert := range alerts {
		if alert.IsFiring() {
			return alert
		}
	}
	return nil
}

// sendNotifications sends notifications to all configured notifiers.
func (uc *ProcessAlertUseCase) sendNotifications(ctx context.Context, alert *entity.Alert, output *dto.ProcessAlertOutput) {
	// Get matching subscribers if subscriber matcher is configured
	var slackUserIDs []string
	var pdSubscribers []service.UseCaseMatchedSubscriber

	if uc.subscriberMatcher != nil {
		// Get Slack subscribers (all matched at once for mentions)
		slackMatched := uc.subscriberMatcher.MatchAlertForSlackUseCase(alert)
		slackUserIDs = service.GetSlackUserIDsFromUseCase(slackMatched)

		if len(slackMatched) > 0 {
			names := make([]string, len(slackMatched))
			for i, m := range slackMatched {
				names[i] = m.Name
			}
			uc.logger.Info("matched subscribers for Slack",
				"alertID", alert.ID,
				"subscribers", names,
			)
		}

		// Get PagerDuty subscribers (ordered by match count for sequential escalation)
		pdSubscribers = uc.subscriberMatcher.MatchAlertForPagerDutyUseCase(alert)

		if len(pdSubscribers) > 0 {
			names := make([]string, len(pdSubscribers))
			for i, m := range pdSubscribers {
				names[i] = fmt.Sprintf("%s(%d)", m.Name, m.MatchCount)
			}
			uc.logger.Info("matched subscribers for PagerDuty",
				"alertID", alert.ID,
				"subscribers", names,
			)
		}
	}

	for _, notifier := range uc.notifiers {
		var messageID string
		var err error

		switch notifier.Name() {
		case "slack":
			messageID, err = uc.sendSlackNotification(ctx, alert, slackUserIDs)
		case "pagerduty":
			messageID, err = uc.sendPagerDutyNotification(ctx, alert, pdSubscribers)
		default:
			// Use generic notifier for other notification types
			messageID, err = notifier.Notify(ctx, alert)
		}

		if err != nil {
			uc.logger.Error("notification failed",
				"notifier", notifier.Name(),
				"alertID", alert.ID,
				"error", err,
			)
			output.NotificationsFailed = append(output.NotificationsFailed, dto.NotificationError{
				NotifierName: notifier.Name(),
				Error:        err,
			})
			continue
		}

		// Store message ID for later updates
		uc.storeMessageID(ctx, alert, notifier.Name(), messageID)
		output.NotificationsSent = append(output.NotificationsSent, notifier.Name())

		uc.logger.Info("notification sent",
			"notifier", notifier.Name(),
			"alertID", alert.ID,
			"messageID", messageID,
		)
	}
}

// sendSlackNotification sends a Slack notification with optional user mentions.
func (uc *ProcessAlertUseCase) sendSlackNotification(ctx context.Context, alert *entity.Alert, slackUserIDs []string) (string, error) {
	// Use subscriber-aware notifier if available and we have matching subscribers
	if uc.slackNotifier != nil && len(slackUserIDs) > 0 {
		return uc.slackNotifier.NotifyWithMentions(ctx, alert, slackUserIDs)
	}

	// Fallback to the generic Slack notifier from the notifiers list
	for _, notifier := range uc.notifiers {
		if notifier.Name() == "slack" {
			return notifier.Notify(ctx, alert)
		}
	}

	return "", fmt.Errorf("slack notifier not found")
}

// sendPagerDutyNotification sends PagerDuty notifications to subscribers sequentially.
func (uc *ProcessAlertUseCase) sendPagerDutyNotification(ctx context.Context, alert *entity.Alert, subscribers []service.UseCaseMatchedSubscriber) (string, error) {
	// Use subscriber-aware notifier if available and we have matching subscribers
	if uc.pagerDutyNotifier != nil && len(subscribers) > 0 {
		// Convert to PagerDuty notification format
		pdNotifications := make([]PagerDutySubscriberNotification, len(subscribers))
		for i, sub := range subscribers {
			pdNotifications[i] = PagerDutySubscriberNotification{
				SubscriberName:  sub.Name,
				PagerDutyUserID: sub.PagerDutyUserID,
				RoutingKey:      sub.PagerDutyRoutingKey,
				MatchCount:      sub.MatchCount,
			}
		}

		// Send to each subscriber sequentially (most matches first)
		results := uc.pagerDutyNotifier.NotifySubscribersSequentially(ctx, alert, pdNotifications)

		// Log results for each subscriber
		var firstDedupKey string
		for name, result := range results {
			if len(result) > 6 && result[:6] == "error:" {
				uc.logger.Error("PagerDuty subscriber notification failed",
					"alertID", alert.ID,
					"subscriber", name,
					"error", result,
				)
			} else {
				uc.logger.Info("PagerDuty subscriber notified",
					"alertID", alert.ID,
					"subscriber", name,
					"dedupKey", result,
				)
				if firstDedupKey == "" {
					firstDedupKey = result
				}
			}
		}

		// Return the first dedup key for tracking
		if firstDedupKey != "" {
			return firstDedupKey, nil
		}
	}

	// Fallback to the generic PagerDuty notifier from the notifiers list
	for _, notifier := range uc.notifiers {
		if notifier.Name() == "pagerduty" {
			return notifier.Notify(ctx, alert)
		}
	}

	return "", fmt.Errorf("pagerduty notifier not found")
}

// updateNotifications updates existing notifications for resolved/acked alerts.
func (uc *ProcessAlertUseCase) updateNotifications(ctx context.Context, alert *entity.Alert, output *dto.ProcessAlertOutput) {
	for _, notifier := range uc.notifiers {
		messageID := uc.getMessageID(alert, notifier.Name())
		if messageID == "" {
			continue
		}

		if err := notifier.UpdateMessage(ctx, messageID, alert); err != nil {
			uc.logger.Error("failed to update notification",
				"notifier", notifier.Name(),
				"alertID", alert.ID,
				"messageID", messageID,
				"error", err,
			)
			output.NotificationsFailed = append(output.NotificationsFailed, dto.NotificationError{
				NotifierName: notifier.Name(),
				Error:        err,
			})
			continue
		}

		output.NotificationsSent = append(output.NotificationsSent, notifier.Name())
	}
}

// storeMessageID stores the message ID for a notifier.
func (uc *ProcessAlertUseCase) storeMessageID(ctx context.Context, alert *entity.Alert, notifierName, messageID string) {
	alert.SetExternalReference(notifierName, messageID)

	// Update the alert with the new message ID
	if err := uc.alertRepo.Update(ctx, alert); err != nil {
		uc.logger.Error("failed to store message ID",
			"notifier", notifierName,
			"alertID", alert.ID,
			"error", err,
		)
	}
}

// getMessageID retrieves the message ID for a notifier.
func (uc *ProcessAlertUseCase) getMessageID(alert *entity.Alert, notifierName string) string {
	return alert.GetExternalReference(notifierName)
}
