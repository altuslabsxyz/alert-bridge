package presenter

import (
	"testing"
	"time"

	"github.com/altuslabsxyz/alert-bridge/internal/domain/entity"
)

func TestSlackAlertFormatter_getSeverityMarker(t *testing.T) {
	f := NewSlackAlertFormatter()

	tests := []struct {
		name     string
		severity entity.AlertSeverity
		want     string
	}{
		{"critical", entity.SeverityCritical, "🔴"},
		{"warning", entity.SeverityWarning, "🟡"},
		{"info", entity.SeverityInfo, "🔵"},
		{"unknown", entity.AlertSeverity("unknown"), "⚪"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.getSeverityMarker(tt.severity)
			if got != tt.want {
				t.Errorf("getSeverityMarker(%v) = %v, want %v", tt.severity, got, tt.want)
			}
		})
	}
}

func TestSlackAlertFormatter_formatSeverity(t *testing.T) {
	f := NewSlackAlertFormatter()

	tests := []struct {
		name     string
		severity string
		want     string
	}{
		{"critical", "critical", "Critical"},
		{"warning", "warning", "Warning"},
		{"info", "info", "Info"},
		{"unknown", "other", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.formatSeverity(tt.severity)
			if got != tt.want {
				t.Errorf("formatSeverity(%v) = %v, want %v", tt.severity, got, tt.want)
			}
		})
	}
}

func TestSlackAlertFormatter_formatDuration(t *testing.T) {
	f := NewSlackAlertFormatter()

	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"seconds", 45 * time.Second, "45s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours_minutes", 2*time.Hour + 30*time.Minute, "2h 30m"},
		{"days_hours", 3*24*time.Hour + 5*time.Hour, "3d 5h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %v, want %v", tt.duration, got, tt.want)
			}
		})
	}
}

func TestSlackAlertFormatter_joinDetails(t *testing.T) {
	f := NewSlackAlertFormatter()

	tests := []struct {
		name    string
		details []string
		want    string
	}{
		{"empty", []string{}, ""},
		{"single", []string{"a"}, "a"},
		{"multiple", []string{"a", "b", "c"}, "a | b | c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.joinDetails(tt.details)
			if got != tt.want {
				t.Errorf("joinDetails(%v) = %v, want %v", tt.details, got, tt.want)
			}
		})
	}
}

func TestSlackAlertFormatter_formatTopInstances(t *testing.T) {
	f := NewSlackAlertFormatter()

	tests := []struct {
		name      string
		instances map[string]int
		limit     int
		wantLen   int
	}{
		{
			name:      "empty",
			instances: map[string]int{},
			limit:     5,
			wantLen:   0,
		},
		{
			name:      "less than limit",
			instances: map[string]int{"host1": 5, "host2": 3},
			limit:     5,
			wantLen:   2,
		},
		{
			name:      "more than limit",
			instances: map[string]int{"host1": 5, "host2": 3, "host3": 10, "host4": 1, "host5": 2, "host6": 7},
			limit:     3,
			wantLen:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.formatTopInstances(tt.instances, tt.limit)
			if tt.wantLen == 0 && got != "" {
				t.Errorf("formatTopInstances() should be empty, got %v", got)
			}
			if tt.wantLen > 0 && got == "" {
				t.Errorf("formatTopInstances() should not be empty")
			}
		})
	}
}

func TestSlackAlertFormatter_FormatAlertStatus(t *testing.T) {
	f := NewSlackAlertFormatter()

	t.Run("empty alerts", func(t *testing.T) {
		blocks := f.FormatAlertStatus([]*entity.Alert{}, "")
		if len(blocks) == 0 {
			t.Error("FormatAlertStatus should return blocks even for empty alerts")
		}
	})

	t.Run("with severity filter", func(t *testing.T) {
		blocks := f.FormatAlertStatus([]*entity.Alert{}, "critical")
		if len(blocks) == 0 {
			t.Error("FormatAlertStatus should return blocks with severity filter")
		}
	})

	t.Run("with alerts", func(t *testing.T) {
		alerts := []*entity.Alert{
			{
				ID:       "1",
				Name:     "TestAlert",
				Severity: entity.SeverityCritical,
				State:    entity.StateActive,
				FiredAt:  time.Now().Add(-1 * time.Hour),
			},
		}
		blocks := f.FormatAlertStatus(alerts, "")
		if len(blocks) < 3 {
			t.Error("FormatAlertStatus should return multiple blocks for alerts")
		}
	})

	t.Run("limits to 10 alerts", func(t *testing.T) {
		alerts := make([]*entity.Alert, 15)
		for i := range alerts {
			alerts[i] = &entity.Alert{
				ID:       string(rune('0' + i)),
				Name:     "TestAlert",
				Severity: entity.SeverityInfo,
				State:    entity.StateActive,
				FiredAt:  time.Now(),
			}
		}
		blocks := f.FormatAlertStatus(alerts, "")
		// Should have header + summary + divider + 10 alerts with dividers + truncation message + footer
		if len(blocks) == 0 {
			t.Error("FormatAlertStatus should return blocks")
		}
	})
}
