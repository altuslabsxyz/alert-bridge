package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/altuslabsxyz/alert-bridge/internal/domain/entity"
	"github.com/altuslabsxyz/alert-bridge/internal/infrastructure/config"
)

func boolPtr(b bool) *bool {
	return &b
}

func TestSubscriberMatcher_MatchAlert(t *testing.T) {
	tests := []struct {
		name           string
		subscribers    []config.SubscriberConfig
		alertLabels    map[string]string
		expectedNames  []string
		expectedCounts []int
	}{
		{
			name: "single subscriber matches all labels",
			subscribers: []config.SubscriberConfig{
				{
					Name:        "jinu",
					SlackUserID: "U123",
					Labels:      map[string]string{"chain": "axelar"},
				},
			},
			alertLabels:    map[string]string{"chain": "axelar", "severity": "critical"},
			expectedNames:  []string{"jinu"},
			expectedCounts: []int{1},
		},
		{
			name: "subscriber with multiple labels - all match",
			subscribers: []config.SubscriberConfig{
				{
					Name:        "jinu",
					SlackUserID: "U123",
					Labels:      map[string]string{"chain": "axelar", "severity": "critical"},
				},
			},
			alertLabels:    map[string]string{"chain": "axelar", "severity": "critical", "env": "prod"},
			expectedNames:  []string{"jinu"},
			expectedCounts: []int{2},
		},
		{
			name: "subscriber with multiple labels - partial match (no match)",
			subscribers: []config.SubscriberConfig{
				{
					Name:        "jinu",
					SlackUserID: "U123",
					Labels:      map[string]string{"chain": "axelar", "severity": "critical"},
				},
			},
			alertLabels:    map[string]string{"chain": "axelar", "severity": "warning"},
			expectedNames:  []string{},
			expectedCounts: []int{},
		},
		{
			name: "multiple subscribers - sorted by match count",
			subscribers: []config.SubscriberConfig{
				{
					Name:        "oncall",
					SlackUserID: "U100",
					Labels:      map[string]string{"severity": "critical"},
				},
				{
					Name:        "jinu",
					SlackUserID: "U123",
					Labels:      map[string]string{"chain": "axelar", "severity": "critical", "env": "prod"},
				},
				{
					Name:        "jeseon",
					SlackUserID: "U456",
					Labels:      map[string]string{"chain": "axelar", "severity": "critical"},
				},
			},
			alertLabels: map[string]string{
				"chain":    "axelar",
				"severity": "critical",
				"env":      "prod",
				"region":   "ap-northeast-1",
			},
			expectedNames:  []string{"jinu", "jeseon", "oncall"},
			expectedCounts: []int{3, 2, 1},
		},
		{
			name: "disabled subscriber is excluded",
			subscribers: []config.SubscriberConfig{
				{
					Name:        "jinu",
					SlackUserID: "U123",
					Labels:      map[string]string{"chain": "axelar"},
					Enabled:     boolPtr(false),
				},
				{
					Name:        "jeseon",
					SlackUserID: "U456",
					Labels:      map[string]string{"chain": "axelar"},
					Enabled:     boolPtr(true),
				},
			},
			alertLabels:    map[string]string{"chain": "axelar"},
			expectedNames:  []string{"jeseon"},
			expectedCounts: []int{1},
		},
		{
			name: "subscriber without Enabled field defaults to enabled",
			subscribers: []config.SubscriberConfig{
				{
					Name:        "jinu",
					SlackUserID: "U123",
					Labels:      map[string]string{"chain": "axelar"},
					// Enabled is nil (not set)
				},
			},
			alertLabels:    map[string]string{"chain": "axelar"},
			expectedNames:  []string{"jinu"},
			expectedCounts: []int{1},
		},
		{
			name: "no matching subscribers",
			subscribers: []config.SubscriberConfig{
				{
					Name:        "jinu",
					SlackUserID: "U123",
					Labels:      map[string]string{"chain": "axelar"},
				},
			},
			alertLabels:    map[string]string{"chain": "osmosis"},
			expectedNames:  []string{},
			expectedCounts: []int{},
		},
		{
			name:           "empty subscriber list",
			subscribers:    []config.SubscriberConfig{},
			alertLabels:    map[string]string{"chain": "axelar"},
			expectedNames:  []string{},
			expectedCounts: []int{},
		},
		{
			name: "subscriber with empty labels (never matches)",
			subscribers: []config.SubscriberConfig{
				{
					Name:        "jinu",
					SlackUserID: "U123",
					Labels:      map[string]string{},
				},
			},
			alertLabels:    map[string]string{"chain": "axelar"},
			expectedNames:  []string{},
			expectedCounts: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewSubscriberMatcher(tt.subscribers)
			alert := &entity.Alert{Labels: tt.alertLabels}

			matched := matcher.MatchAlert(alert)

			require.Len(t, matched, len(tt.expectedNames))

			for i, m := range matched {
				assert.Equal(t, tt.expectedNames[i], m.Subscriber.Name, "subscriber name at index %d", i)
				assert.Equal(t, tt.expectedCounts[i], m.MatchCount, "match count at index %d", i)
			}
		})
	}
}

func TestGetSlackUserIDs(t *testing.T) {
	matched := []MatchedSubscriber{
		{Subscriber: config.SubscriberConfig{Name: "jinu", SlackUserID: "U123"}, MatchCount: 3},
		{Subscriber: config.SubscriberConfig{Name: "jeseon", SlackUserID: "U456"}, MatchCount: 2},
		{Subscriber: config.SubscriberConfig{Name: "duplicate", SlackUserID: "U123"}, MatchCount: 1}, // duplicate
		{Subscriber: config.SubscriberConfig{Name: "no-slack", SlackUserID: ""}, MatchCount: 1},
	}

	userIDs := GetSlackUserIDs(matched)

	assert.Equal(t, []string{"U123", "U456"}, userIDs)
}

func TestGetPagerDutyUserIDs(t *testing.T) {
	matched := []MatchedSubscriber{
		{Subscriber: config.SubscriberConfig{Name: "jinu", PagerDutyUserID: "PD123"}, MatchCount: 3},
		{Subscriber: config.SubscriberConfig{Name: "jeseon", PagerDutyUserID: "PD456"}, MatchCount: 2},
		{Subscriber: config.SubscriberConfig{Name: "oncall", PagerDutyUserID: "PD789"}, MatchCount: 1},
	}

	userIDs := GetPagerDutyUserIDs(matched)

	// Should be in order by match count (highest first)
	assert.Equal(t, []string{"PD123", "PD456", "PD789"}, userIDs)
}

func TestGetPagerDutyRoutingKeys(t *testing.T) {
	matched := []MatchedSubscriber{
		{Subscriber: config.SubscriberConfig{Name: "jinu", PagerDutyRoutingKey: "RK123"}, MatchCount: 3},
		{Subscriber: config.SubscriberConfig{Name: "jeseon", PagerDutyRoutingKey: ""}, MatchCount: 2},
		{Subscriber: config.SubscriberConfig{Name: "oncall", PagerDutyRoutingKey: "RK789"}, MatchCount: 1},
	}

	routingKeys := GetPagerDutyRoutingKeys(matched)

	assert.Len(t, routingKeys, 2)
	assert.Equal(t, "RK123", routingKeys["jinu"])
	assert.Equal(t, "RK789", routingKeys["oncall"])
	_, exists := routingKeys["jeseon"]
	assert.False(t, exists)
}

func TestSubscriberMatcher_UpdateSubscribers(t *testing.T) {
	initialSubs := []config.SubscriberConfig{
		{Name: "jinu", SlackUserID: "U123", Labels: map[string]string{"chain": "axelar"}},
	}
	matcher := NewSubscriberMatcher(initialSubs)

	alert := &entity.Alert{Labels: map[string]string{"chain": "axelar"}}
	matched := matcher.MatchAlert(alert)
	require.Len(t, matched, 1)
	assert.Equal(t, "jinu", matched[0].Subscriber.Name)

	// Update subscribers
	newSubs := []config.SubscriberConfig{
		{Name: "jeseon", SlackUserID: "U456", Labels: map[string]string{"chain": "axelar"}},
	}
	matcher.UpdateSubscribers(newSubs)

	matched = matcher.MatchAlert(alert)
	require.Len(t, matched, 1)
	assert.Equal(t, "jeseon", matched[0].Subscriber.Name)
}
