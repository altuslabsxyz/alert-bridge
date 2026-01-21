package service

import (
	"sort"

	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/entity"
	"github.com/qj0r9j0vc2/alert-bridge/internal/infrastructure/config"
)

// MatchedSubscriber represents a subscriber that matched an alert along with
// the number of labels that matched.
type MatchedSubscriber struct {
	// Subscriber is the matched subscriber configuration.
	Subscriber config.SubscriberConfig

	// MatchCount is the number of labels that matched between the subscriber's
	// filter and the alert's labels.
	MatchCount int
}

// SubscriberMatcher matches alerts to subscribers based on label filters.
type SubscriberMatcher struct {
	subscribers []config.SubscriberConfig
}

// NewSubscriberMatcher creates a new SubscriberMatcher with the given subscribers.
func NewSubscriberMatcher(subscribers []config.SubscriberConfig) *SubscriberMatcher {
	return &SubscriberMatcher{
		subscribers: subscribers,
	}
}

// UpdateSubscribers updates the subscriber list (for config hot-reload).
func (m *SubscriberMatcher) UpdateSubscribers(subscribers []config.SubscriberConfig) {
	m.subscribers = subscribers
}

// MatchAlert finds all subscribers that match the given alert's labels.
// Returns subscribers sorted by match count in descending order (most matches first).
// A subscriber matches if ALL of their configured labels exist in the alert with the same values.
func (m *SubscriberMatcher) MatchAlert(alert *entity.Alert) []MatchedSubscriber {
	var matched []MatchedSubscriber

	for _, sub := range m.subscribers {
		if !sub.IsEnabled() {
			continue
		}

		matchCount := m.countMatchingLabels(sub.Labels, alert.Labels)
		if matchCount > 0 && matchCount == len(sub.Labels) {
			// All subscriber labels matched
			matched = append(matched, MatchedSubscriber{
				Subscriber: sub,
				MatchCount: matchCount,
			})
		}
	}

	// Sort by match count descending (most matches first)
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].MatchCount > matched[j].MatchCount
	})

	return matched
}

// MatchAlertForSlack returns all matching subscribers for Slack mentions.
// Returns all subscribers that match, regardless of order (they'll all be mentioned).
func (m *SubscriberMatcher) MatchAlertForSlack(alert *entity.Alert) []MatchedSubscriber {
	return m.MatchAlert(alert)
}

// MatchAlertForPagerDuty returns matching subscribers ordered by match count
// for sequential PagerDuty escalation (most matches first).
func (m *SubscriberMatcher) MatchAlertForPagerDuty(alert *entity.Alert) []MatchedSubscriber {
	return m.MatchAlert(alert)
}

// countMatchingLabels counts how many labels from the filter match the alert labels.
func (m *SubscriberMatcher) countMatchingLabels(filterLabels, alertLabels map[string]string) int {
	if len(filterLabels) == 0 {
		return 0
	}

	count := 0
	for key, value := range filterLabels {
		if alertValue, exists := alertLabels[key]; exists && alertValue == value {
			count++
		}
	}
	return count
}

// GetSlackUserIDs returns a list of Slack user IDs for the matched subscribers.
func GetSlackUserIDs(matched []MatchedSubscriber) []string {
	var userIDs []string
	seen := make(map[string]bool)

	for _, m := range matched {
		if m.Subscriber.SlackUserID != "" && !seen[m.Subscriber.SlackUserID] {
			userIDs = append(userIDs, m.Subscriber.SlackUserID)
			seen[m.Subscriber.SlackUserID] = true
		}
	}
	return userIDs
}

// GetPagerDutyUserIDs returns a list of PagerDuty user IDs for the matched subscribers,
// ordered by match count (most matches first for escalation priority).
func GetPagerDutyUserIDs(matched []MatchedSubscriber) []string {
	var userIDs []string
	seen := make(map[string]bool)

	for _, m := range matched {
		if m.Subscriber.PagerDutyUserID != "" && !seen[m.Subscriber.PagerDutyUserID] {
			userIDs = append(userIDs, m.Subscriber.PagerDutyUserID)
			seen[m.Subscriber.PagerDutyUserID] = true
		}
	}
	return userIDs
}

// GetPagerDutyRoutingKeys returns a map of subscriber names to their PagerDuty routing keys.
// Only includes subscribers that have a custom routing key configured.
func GetPagerDutyRoutingKeys(matched []MatchedSubscriber) map[string]string {
	routingKeys := make(map[string]string)

	for _, m := range matched {
		if m.Subscriber.PagerDutyRoutingKey != "" {
			routingKeys[m.Subscriber.Name] = m.Subscriber.PagerDutyRoutingKey
		}
	}
	return routingKeys
}

// UseCaseMatchedSubscriber is an alias for use by the alert use case layer.
// This provides the necessary fields without depending on config types.
type UseCaseMatchedSubscriber struct {
	Name                string
	SlackUserID         string
	PagerDutyUserID     string
	PagerDutyRoutingKey string
	MatchCount          int
}

// ToUseCaseFormat converts matched subscribers to use case format.
func ToUseCaseFormat(matched []MatchedSubscriber) []UseCaseMatchedSubscriber {
	result := make([]UseCaseMatchedSubscriber, len(matched))
	for i, m := range matched {
		result[i] = UseCaseMatchedSubscriber{
			Name:                m.Subscriber.Name,
			SlackUserID:         m.Subscriber.SlackUserID,
			PagerDutyUserID:     m.Subscriber.PagerDutyUserID,
			PagerDutyRoutingKey: m.Subscriber.PagerDutyRoutingKey,
			MatchCount:          m.MatchCount,
		}
	}
	return result
}

// MatchAlertForSlackUseCase returns matching subscribers in use case format for Slack mentions.
func (m *SubscriberMatcher) MatchAlertForSlackUseCase(alert *entity.Alert) []UseCaseMatchedSubscriber {
	return ToUseCaseFormat(m.MatchAlertForSlack(alert))
}

// MatchAlertForPagerDutyUseCase returns matching subscribers in use case format for PagerDuty escalation.
func (m *SubscriberMatcher) MatchAlertForPagerDutyUseCase(alert *entity.Alert) []UseCaseMatchedSubscriber {
	return ToUseCaseFormat(m.MatchAlertForPagerDuty(alert))
}

// GetSlackUserIDsFromUseCase returns a list of Slack user IDs for the matched subscribers.
func GetSlackUserIDsFromUseCase(matched []UseCaseMatchedSubscriber) []string {
	var userIDs []string
	seen := make(map[string]bool)

	for _, m := range matched {
		if m.SlackUserID != "" && !seen[m.SlackUserID] {
			userIDs = append(userIDs, m.SlackUserID)
			seen[m.SlackUserID] = true
		}
	}
	return userIDs
}
