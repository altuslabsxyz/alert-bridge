package alert

import (
	"context"

	"github.com/altuslabsxyz/alert-bridge/internal/domain/entity"
	"github.com/altuslabsxyz/alert-bridge/internal/domain/logger"
)

// Notifier defines the contract for sending alert notifications.
// OCP: New notification channels implement this interface.
type Notifier interface {
	// Notify sends an alert to the notification channel.
	// Returns a channel-specific message ID for tracking.
	Notify(ctx context.Context, alert *entity.Alert) (messageID string, err error)

	// UpdateMessage updates an existing notification (e.g., after ack or resolve).
	UpdateMessage(ctx context.Context, messageID string, alert *entity.Alert) error

	// Name returns the notifier identifier (e.g., "slack", "pagerduty").
	Name() string
}

// SlackSubscriberNotifier extends Notifier with subscriber mention support.
// This interface is implemented by the Slack client to support @mentioning
// matching subscribers when sending alerts.
type SlackSubscriberNotifier interface {
	Notifier

	// NotifyWithMentions sends an alert to Slack with @mentions for the given user IDs.
	// All matching subscribers are mentioned at once in the message.
	NotifyWithMentions(ctx context.Context, alert *entity.Alert, slackUserIDs []string) (messageID string, err error)
}

// PagerDutySubscriberNotification represents a notification to be sent to a specific subscriber.
type PagerDutySubscriberNotification struct {
	// SubscriberName is the human-readable name of the subscriber.
	SubscriberName string

	// PagerDutyUserID is the PagerDuty user ID to target.
	PagerDutyUserID string

	// RoutingKey is the routing key to use for this subscriber.
	// If empty, the default routing key will be used.
	RoutingKey string

	// MatchCount indicates how many labels matched for priority ordering.
	MatchCount int
}

// PagerDutySubscriberNotifier extends Notifier with subscriber-aware escalation.
// This interface is implemented by the PagerDuty client to support sequential
// notification of subscribers based on label match priority.
type PagerDutySubscriberNotifier interface {
	Notifier

	// NotifySubscribersSequentially sends alerts to subscribers in order of match priority.
	// Returns a map of subscriber names to their dedup keys (or error messages).
	NotifySubscribersSequentially(ctx context.Context, alert *entity.Alert, subscribers []PagerDutySubscriberNotification) map[string]string
}

// SubscriberMatcher matches alerts to subscribers based on label filters.
type SubscriberMatcher interface {
	// MatchAlertForSlack returns subscribers matched for Slack mentions.
	MatchAlertForSlack(alert *entity.Alert) []MatchedSubscriber

	// MatchAlertForPagerDuty returns subscribers matched for PagerDuty escalation,
	// ordered by match count (most matches first).
	MatchAlertForPagerDuty(alert *entity.Alert) []MatchedSubscriber
}

// MatchedSubscriber represents a subscriber that matched an alert.
type MatchedSubscriber struct {
	Name                string
	SlackUserID         string
	PagerDutyUserID     string
	PagerDutyRoutingKey string
	MatchCount          int
}

// Logger is the unified logging interface from domain layer.
type Logger = logger.Logger
