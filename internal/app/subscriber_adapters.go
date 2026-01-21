package app

import (
	"context"

	"github.com/altuslabsxyz/alert-bridge/internal/domain/entity"
	"github.com/altuslabsxyz/alert-bridge/internal/infrastructure/pagerduty"
	"github.com/altuslabsxyz/alert-bridge/internal/infrastructure/slack"
	"github.com/altuslabsxyz/alert-bridge/internal/usecase/alert"
)

// SlackSubscriberNotifierAdapter adapts the Slack client to the SlackSubscriberNotifier interface.
type SlackSubscriberNotifierAdapter struct {
	client *slack.Client
}

// NewSlackSubscriberNotifierAdapter creates a new adapter.
func NewSlackSubscriberNotifierAdapter(client *slack.Client) *SlackSubscriberNotifierAdapter {
	return &SlackSubscriberNotifierAdapter{client: client}
}

// Notify sends an alert to Slack.
func (a *SlackSubscriberNotifierAdapter) Notify(ctx context.Context, alertEntity *entity.Alert) (string, error) {
	return a.client.Notify(ctx, alertEntity)
}

// UpdateMessage updates an existing Slack message.
func (a *SlackSubscriberNotifierAdapter) UpdateMessage(ctx context.Context, messageID string, alertEntity *entity.Alert) error {
	return a.client.UpdateMessage(ctx, messageID, alertEntity)
}

// Name returns the notifier identifier.
func (a *SlackSubscriberNotifierAdapter) Name() string {
	return a.client.Name()
}

// NotifyWithMentions sends an alert to Slack with @mentions for the given user IDs.
func (a *SlackSubscriberNotifierAdapter) NotifyWithMentions(ctx context.Context, alertEntity *entity.Alert, slackUserIDs []string) (string, error) {
	return a.client.NotifyWithMentions(ctx, alertEntity, slackUserIDs)
}

// PagerDutySubscriberNotifierAdapter adapts the PagerDuty client to the PagerDutySubscriberNotifier interface.
type PagerDutySubscriberNotifierAdapter struct {
	client *pagerduty.Client
}

// NewPagerDutySubscriberNotifierAdapter creates a new adapter.
func NewPagerDutySubscriberNotifierAdapter(client *pagerduty.Client) *PagerDutySubscriberNotifierAdapter {
	return &PagerDutySubscriberNotifierAdapter{client: client}
}

// Notify sends an alert to PagerDuty.
func (a *PagerDutySubscriberNotifierAdapter) Notify(ctx context.Context, alertEntity *entity.Alert) (string, error) {
	return a.client.Notify(ctx, alertEntity)
}

// UpdateMessage updates an existing PagerDuty incident.
func (a *PagerDutySubscriberNotifierAdapter) UpdateMessage(ctx context.Context, messageID string, alertEntity *entity.Alert) error {
	return a.client.UpdateMessage(ctx, messageID, alertEntity)
}

// Name returns the notifier identifier.
func (a *PagerDutySubscriberNotifierAdapter) Name() string {
	return a.client.Name()
}

// NotifySubscribersSequentially sends PagerDuty alerts to subscribers sequentially.
func (a *PagerDutySubscriberNotifierAdapter) NotifySubscribersSequentially(ctx context.Context, alertEntity *entity.Alert, subscribers []alert.PagerDutySubscriberNotification) map[string]string {
	// Convert alert.PagerDutySubscriberNotification to pagerduty.SubscriberNotification
	pdSubscribers := make([]pagerduty.SubscriberNotification, len(subscribers))
	for i, sub := range subscribers {
		pdSubscribers[i] = pagerduty.SubscriberNotification{
			SubscriberName:  sub.SubscriberName,
			PagerDutyUserID: sub.PagerDutyUserID,
			RoutingKey:      sub.RoutingKey,
			MatchCount:      sub.MatchCount,
		}
	}
	return a.client.NotifySubscribersSequentially(ctx, alertEntity, pdSubscribers)
}

// Verify interface implementations at compile time
var _ alert.SlackSubscriberNotifier = (*SlackSubscriberNotifierAdapter)(nil)
var _ alert.PagerDutySubscriberNotifier = (*PagerDutySubscriberNotifierAdapter)(nil)
