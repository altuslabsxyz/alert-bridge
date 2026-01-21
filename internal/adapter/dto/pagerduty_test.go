package dto

import "testing"

func TestIsSupportedEventType(t *testing.T) {
	tests := []struct {
		eventType string
		expected  bool
	}{
		// Supported event types
		{"incident.acknowledged", true},
		{"incident.resolved", true},
		{"incident.unacknowledged", true},
		{"incident.reassigned", true},

		// Unsupported event types
		{"incident.triggered", false},
		{"incident.escalated", false},
		{"incident.delegated", false},
		{"incident.annotated", false},
		{"service.created", false},
		{"service.updated", false},

		// Edge cases
		{"", false},
		{"incident", false},
		{"acknowledged", false},
		{"INCIDENT.ACKNOWLEDGED", false},  // case sensitive
		{"incident.acknowledged ", false}, // trailing space
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			if got := IsSupportedEventType(tt.eventType); got != tt.expected {
				t.Errorf("IsSupportedEventType(%q) = %v, want %v", tt.eventType, got, tt.expected)
			}
		})
	}
}
