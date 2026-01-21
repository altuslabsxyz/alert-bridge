package slack

import (
	"strings"
	"testing"
	"time"

	"github.com/altuslabsxyz/alert-bridge/internal/domain/entity"
)

func TestFormatSlackTime(t *testing.T) {
	// Fixed test time: 2024-01-21 15:30:45 UTC
	testTime := time.Date(2024, 1, 21, 15, 30, 45, 0, time.UTC)

	tests := []struct {
		name    string
		time    time.Time
		format  string
		wantFmt string // expected format pattern in output
	}{
		{
			name:    "date short format",
			time:    testTime,
			format:  SlackDateShort,
			wantFmt: "<!date^1705851045^{date_short} {time}|2024-01-21 15:30 UTC>",
		},
		{
			name:    "time only format",
			time:    testTime,
			format:  SlackTimeOnly,
			wantFmt: "<!date^1705851045^{time}|2024-01-21 15:30 UTC>",
		},
		{
			name:    "date full format",
			time:    testTime,
			format:  SlackDateFull,
			wantFmt: "<!date^1705851045^{date} {time}|2024-01-21 15:30 UTC>",
		},
		{
			name:    "date long format",
			time:    testTime,
			format:  SlackDateLong,
			wantFmt: "<!date^1705851045^{date_long} {time}|2024-01-21 15:30 UTC>",
		},
		{
			name:    "date short only format",
			time:    testTime,
			format:  SlackDateShortOnly,
			wantFmt: "<!date^1705851045^{date_short}|2024-01-21 15:30 UTC>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSlackTime(tt.time, tt.format)
			if got != tt.wantFmt {
				t.Errorf("FormatSlackTime() = %q, want %q", got, tt.wantFmt)
			}
		})
	}
}

func TestFormatSlackTime_DifferentTimezones(t *testing.T) {
	// Test that the same instant in different timezones produces the same Unix timestamp
	utcTime := time.Date(2024, 1, 21, 15, 0, 0, 0, time.UTC)

	// Load different timezone
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Skip("Asia/Tokyo timezone not available")
	}

	// Same instant but in JST (2024-01-22 00:00:00 JST = 2024-01-21 15:00:00 UTC)
	jstTime := utcTime.In(jst)

	utcResult := FormatSlackTime(utcTime, SlackDateShort)
	jstResult := FormatSlackTime(jstTime, SlackDateShort)

	// Both should produce the same output since they represent the same instant
	if utcResult != jstResult {
		t.Errorf("Same instant should produce same output.\nUTC: %s\nJST: %s", utcResult, jstResult)
	}
}

func TestNewMessageBuilder(t *testing.T) {
	t.Run("with nil durations uses defaults", func(t *testing.T) {
		builder := NewMessageBuilder(nil)
		if builder == nil {
			t.Fatal("NewMessageBuilder(nil) returned nil")
		}
		if len(builder.silenceDurations) != 4 {
			t.Errorf("expected 4 default durations, got %d", len(builder.silenceDurations))
		}
		// Check default values
		expected := []time.Duration{15 * time.Minute, 1 * time.Hour, 4 * time.Hour, 24 * time.Hour}
		for i, exp := range expected {
			if builder.silenceDurations[i] != exp {
				t.Errorf("silenceDurations[%d] = %v, want %v", i, builder.silenceDurations[i], exp)
			}
		}
	})

	t.Run("with empty slice uses defaults", func(t *testing.T) {
		builder := NewMessageBuilder([]time.Duration{})
		if len(builder.silenceDurations) != 4 {
			t.Errorf("expected 4 default durations, got %d", len(builder.silenceDurations))
		}
	})

	t.Run("with custom durations", func(t *testing.T) {
		custom := []time.Duration{30 * time.Minute, 2 * time.Hour}
		builder := NewMessageBuilder(custom)
		if len(builder.silenceDurations) != 2 {
			t.Errorf("expected 2 custom durations, got %d", len(builder.silenceDurations))
		}
		if builder.silenceDurations[0] != 30*time.Minute {
			t.Errorf("silenceDurations[0] = %v, want 30m", builder.silenceDurations[0])
		}
	})
}

func TestMessageBuilder_formatDuration(t *testing.T) {
	builder := NewMessageBuilder(nil)

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		// Minutes
		{"1 minute", 1 * time.Minute, "1 min"},
		{"15 minutes", 15 * time.Minute, "15 min"},
		{"30 minutes", 30 * time.Minute, "30 min"},
		{"59 minutes", 59 * time.Minute, "59 min"},

		// Hours
		{"1 hour", 1 * time.Hour, "1 hour"},
		{"2 hours", 2 * time.Hour, "2 hours"},
		{"4 hours", 4 * time.Hour, "4 hours"},
		{"23 hours", 23 * time.Hour, "23 hours"},

		// Days
		{"1 day", 24 * time.Hour, "1 day"},
		{"2 days", 48 * time.Hour, "2 days"},
		{"7 days", 168 * time.Hour, "7 days"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.formatDuration(tt.duration)
			if got != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.expected)
			}
		})
	}
}

func TestMessageBuilder_getStatusInfo(t *testing.T) {
	builder := NewMessageBuilder(nil)

	tests := []struct {
		name      string
		alert     *entity.Alert
		wantEmoji string
		wantText  string
		wantColor string
	}{
		{
			name: "resolved alert",
			alert: &entity.Alert{
				State:    entity.StateResolved,
				Severity: entity.SeverityCritical,
			},
			wantEmoji: "🟢",
			wantText:  "Resolved",
			wantColor: colorResolved,
		},
		{
			name: "acknowledged alert",
			alert: &entity.Alert{
				State:    entity.StateAcked,
				Severity: entity.SeverityCritical,
			},
			wantEmoji: "👀",
			wantText:  "Acknowledged",
			wantColor: colorAcked,
		},
		{
			name: "critical active alert",
			alert: &entity.Alert{
				State:    entity.StateActive,
				Severity: entity.SeverityCritical,
			},
			wantEmoji: "🔴",
			wantText:  "Critical",
			wantColor: colorCritical,
		},
		{
			name: "warning active alert",
			alert: &entity.Alert{
				State:    entity.StateActive,
				Severity: entity.SeverityWarning,
			},
			wantEmoji: "🟡",
			wantText:  "Warning",
			wantColor: colorWarning,
		},
		{
			name: "info active alert",
			alert: &entity.Alert{
				State:    entity.StateActive,
				Severity: entity.SeverityInfo,
			},
			wantEmoji: "🔵",
			wantText:  "Info",
			wantColor: colorInfo,
		},
		{
			name: "unknown severity defaults to info",
			alert: &entity.Alert{
				State:    entity.StateActive,
				Severity: entity.AlertSeverity("unknown"),
			},
			wantEmoji: "🔵",
			wantText:  "Info",
			wantColor: colorInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emoji, text, color := builder.getStatusInfo(tt.alert)
			if emoji != tt.wantEmoji {
				t.Errorf("emoji = %q, want %q", emoji, tt.wantEmoji)
			}
			if text != tt.wantText {
				t.Errorf("text = %q, want %q", text, tt.wantText)
			}
			if color != tt.wantColor {
				t.Errorf("color = %q, want %q", color, tt.wantColor)
			}
		})
	}
}

func TestMessageBuilder_getSeverityBadge(t *testing.T) {
	builder := NewMessageBuilder(nil)

	tests := []struct {
		severity entity.AlertSeverity
		expected string
	}{
		{entity.SeverityCritical, "`CRITICAL`"},
		{entity.SeverityWarning, "`WARNING`"},
		{entity.SeverityInfo, "`INFO`"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			alert := &entity.Alert{Severity: tt.severity}
			got := builder.getSeverityBadge(alert)
			if got != tt.expected {
				t.Errorf("getSeverityBadge() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMessageBuilder_formatState(t *testing.T) {
	builder := NewMessageBuilder(nil)

	tests := []struct {
		state    entity.AlertState
		expected string
	}{
		{entity.StateActive, "🔴 Firing"},
		{entity.StateAcked, "👀 Acknowledged"},
		{entity.StateResolved, "🟢 Resolved"},
		{entity.AlertState("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := builder.formatState(tt.state)
			if got != tt.expected {
				t.Errorf("formatState(%q) = %q, want %q", tt.state, got, tt.expected)
			}
		})
	}
}

func TestMessageBuilder_BuildAlertMessage(t *testing.T) {
	builder := NewMessageBuilder(nil)

	alert := &entity.Alert{
		ID:          "test-alert-123",
		Name:        "HighCPU",
		Summary:     "CPU usage is above 90%",
		Severity:    entity.SeverityCritical,
		State:       entity.StateActive,
		Instance:    "server-01",
		Target:      "cpu",
		Fingerprint: "abc123def456789",
		FiredAt:     time.Now(),
	}

	blocks := builder.BuildAlertMessage(alert)

	if len(blocks) == 0 {
		t.Fatal("BuildAlertMessage() returned empty blocks")
	}
}

func TestMessageBuilder_BuildAckedMessage(t *testing.T) {
	builder := NewMessageBuilder(nil)

	ackedAt := time.Now()
	alert := &entity.Alert{
		ID:       "test-alert-123",
		Name:     "HighCPU",
		Severity: entity.SeverityCritical,
		State:    entity.StateAcked,
		AckedAt:  &ackedAt,
		AckedBy:  "user@example.com",
		FiredAt:  time.Now().Add(-1 * time.Hour),
	}

	blocks := builder.BuildAckedMessage(alert)

	if len(blocks) == 0 {
		t.Fatal("BuildAckedMessage() returned empty blocks")
	}
}

func TestMessageBuilder_BuildResolvedMessage(t *testing.T) {
	builder := NewMessageBuilder(nil)

	resolvedAt := time.Now()
	alert := &entity.Alert{
		ID:         "test-alert-123",
		Name:       "HighCPU",
		Severity:   entity.SeverityCritical,
		State:      entity.StateResolved,
		ResolvedAt: &resolvedAt,
		FiredAt:    time.Now().Add(-2 * time.Hour),
	}

	blocks := builder.BuildResolvedMessage(alert)

	if len(blocks) == 0 {
		t.Fatal("BuildResolvedMessage() returned empty blocks")
	}
}

func TestMessageBuilder_buildActionButtons(t *testing.T) {
	builder := NewMessageBuilder([]time.Duration{15 * time.Minute, 1 * time.Hour})
	alertID := "test-123"

	t.Run("both buttons shown", func(t *testing.T) {
		block := builder.buildActionButtons(alertID, true, true)
		if block == nil {
			t.Fatal("expected action block, got nil")
		}
	})

	t.Run("only ack button", func(t *testing.T) {
		block := builder.buildActionButtons(alertID, true, false)
		if block == nil {
			t.Fatal("expected action block, got nil")
		}
	})

	t.Run("only silence button", func(t *testing.T) {
		block := builder.buildActionButtons(alertID, false, true)
		if block == nil {
			t.Fatal("expected action block, got nil")
		}
	})

	t.Run("no buttons returns nil", func(t *testing.T) {
		block := builder.buildActionButtons(alertID, false, false)
		if block != nil {
			t.Error("expected nil when no buttons shown")
		}
	})
}

func TestMessageBuilder_buildDetailsSection(t *testing.T) {
	builder := NewMessageBuilder(nil)

	t.Run("with all fields", func(t *testing.T) {
		alert := &entity.Alert{
			Instance:    "server-01",
			Target:      "memory",
			Fingerprint: "abc123def456789xyz",
		}
		section := builder.buildDetailsSection(alert)
		if section == nil {
			t.Fatal("expected section, got nil")
		}
	})

	t.Run("with short fingerprint", func(t *testing.T) {
		alert := &entity.Alert{
			Fingerprint: "short",
		}
		section := builder.buildDetailsSection(alert)
		if section == nil {
			t.Fatal("expected section, got nil")
		}
	})

	t.Run("with no optional fields", func(t *testing.T) {
		alert := &entity.Alert{}
		section := builder.buildDetailsSection(alert)
		if section == nil {
			t.Fatal("expected section, got nil")
		}
	})
}

func TestFormatSlackTime_FallbackFormat(t *testing.T) {
	// Test that fallback text is in correct format
	testTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	result := FormatSlackTime(testTime, SlackDateShort)

	// Check fallback format is correct
	if !strings.Contains(result, "2024-06-15 10:30 UTC") {
		t.Errorf("expected fallback format in result, got %s", result)
	}
}

func TestFormatSlackTime_ZeroTime(t *testing.T) {
	// Test with zero time
	result := FormatSlackTime(time.Time{}, SlackDateShort)
	if !strings.HasPrefix(result, "<!date^") {
		t.Errorf("expected Slack date format, got %s", result)
	}
}
