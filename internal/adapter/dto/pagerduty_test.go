package dto

import "testing"

func TestIsSupportedEventType(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		want      bool
	}{
		{"acknowledged", "incident.acknowledged", true},
		{"resolved", "incident.resolved", true},
		{"unacknowledged", "incident.unacknowledged", true},
		{"reassigned", "incident.reassigned", true},
		{"triggered", "incident.triggered", false},
		{"empty", "", false},
		{"invalid", "invalid.event", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSupportedEventType(tt.eventType); got != tt.want {
				t.Errorf("IsSupportedEventType(%q) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}
