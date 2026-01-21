package dto

import (
	"testing"
	"time"
)

func TestSlackCommandDTO_PeriodFilter(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected time.Duration
	}{
		// Empty and all cases
		{"empty text returns 0", "", 0},
		{"all returns 0", "all", 0},

		// Alias cases
		{"today returns 24h", "today", 24 * time.Hour},
		{"week returns 7 days", "week", 7 * 24 * time.Hour},
		{"thisweek returns 7 days", "thisweek", 7 * 24 * time.Hour},

		// Minutes
		{"30m returns 30 minutes", "30m", 30 * time.Minute},
		{"1m returns 1 minute", "1m", 1 * time.Minute},
		{"60m returns 60 minutes", "60m", 60 * time.Minute},

		// Hours
		{"1h returns 1 hour", "1h", 1 * time.Hour},
		{"24h returns 24 hours", "24h", 24 * time.Hour},
		{"48h returns 48 hours", "48h", 48 * time.Hour},

		// Days
		{"1d returns 24 hours", "1d", 24 * time.Hour},
		{"7d returns 7 days", "7d", 7 * 24 * time.Hour},
		{"30d returns 30 days", "30d", 30 * 24 * time.Hour},

		// Weeks
		{"1w returns 7 days", "1w", 7 * 24 * time.Hour},
		{"2w returns 14 days", "2w", 14 * 24 * time.Hour},
		{"4w returns 28 days", "4w", 28 * 24 * time.Hour},

		// Case insensitivity
		{"TODAY returns 24h", "TODAY", 24 * time.Hour},
		{"1H returns 1 hour", "1H", 1 * time.Hour},
		{"WEEK returns 7 days", "WEEK", 7 * 24 * time.Hour},

		// Whitespace handling
		{"  1h  (with spaces) returns 1 hour", "  1h  ", 1 * time.Hour},
		{"\\t24h\\t (with tabs) returns 24 hours", "\t24h\t", 24 * time.Hour},

		// Invalid formats return 0
		{"invalid text returns 0", "invalid", 0},
		{"1x (unknown unit) returns 0", "1x", 0},
		{"h1 (reversed) returns 0", "h1", 0},
		{"1.5h (decimal) returns 0", "1.5h", 0},
		{"-1h (negative) returns 0", "-1h", 0},
		{"0h (zero) returns 0", "0h", 0},
		{"0m (zero minutes) returns 0", "0m", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto := &SlackCommandDTO{Text: tt.text}
			if got := dto.PeriodFilter(); got != tt.expected {
				t.Errorf("PeriodFilter() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSlackCommandDTO_PeriodDescription(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"empty text returns all time", "", "all time"},
		{"all returns all time", "all", "all time"},
		{"1h returns last 1 hour(s)", "1h", "last 1 hour(s)"},
		{"24h converts to days", "24h", "last 1 day(s)"}, // 24 hours = 1 day
		{"1d returns last 1 day(s)", "1d", "last 1 day(s)"},
		{"7d converts to weeks", "7d", "last 1 week(s)"}, // 7 days = 1 week
		{"1w returns last 1 week(s)", "1w", "last 1 week(s)"},
		{"2w returns last 2 week(s)", "2w", "last 2 week(s)"},
		{"3d returns last 3 day(s)", "3d", "last 3 day(s)"}, // < 7 days stays as days
		{"30m returns duration string", "30m", "30m0s"},
		{"invalid returns all time", "invalid", "all time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto := &SlackCommandDTO{Text: tt.text}
			if got := dto.PeriodDescription(); got != tt.expected {
				t.Errorf("PeriodDescription() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSlackCommandDTO_ParseSilenceRequest(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		userID         string
		userName       string
		triggerID      string
		expectedAction SilenceAction
		expectedID     string // for delete action
		validateExtra  func(t *testing.T, req *SilenceRequest)
	}{
		{
			name:           "empty text defaults to list",
			text:           "",
			expectedAction: SilenceActionList,
		},
		{
			name:           "list action",
			text:           "list",
			expectedAction: SilenceActionList,
		},
		{
			name:           "LIST action (case insensitive)",
			text:           "LIST",
			expectedAction: SilenceActionList,
		},
		{
			name:           "create action opens modal",
			text:           "create",
			expectedAction: SilenceActionOpenModal,
		},
		{
			name:           "delete action without ID",
			text:           "delete",
			expectedAction: SilenceActionDelete,
			expectedID:     "",
		},
		{
			name:           "delete action with ID",
			text:           "delete abc123",
			expectedAction: SilenceActionDelete,
			expectedID:     "abc123",
		},
		{
			name:           "delete action with UUID",
			text:           "delete 550e8400-e29b-41d4-a716-446655440000",
			expectedAction: SilenceActionDelete,
			expectedID:     "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:           "duration as first arg opens modal",
			text:           "1h",
			expectedAction: SilenceActionOpenModal,
			validateExtra: func(t *testing.T, req *SilenceRequest) {
				if req.Duration != 1*time.Hour {
					t.Errorf("Duration = %v, want %v", req.Duration, 1*time.Hour)
				}
			},
		},
		{
			name:           "2d duration opens modal with 2 days",
			text:           "2d",
			expectedAction: SilenceActionOpenModal,
			validateExtra: func(t *testing.T, req *SilenceRequest) {
				expected := 2 * 24 * time.Hour
				if req.Duration != expected {
					t.Errorf("Duration = %v, want %v", req.Duration, expected)
				}
			},
		},
		{
			name:           "unknown action defaults to list",
			text:           "unknown",
			expectedAction: SilenceActionList,
		},
		{
			name:           "preserves user info",
			text:           "list",
			userID:         "U12345",
			userName:       "testuser",
			triggerID:      "T67890",
			expectedAction: SilenceActionList,
			validateExtra: func(t *testing.T, req *SilenceRequest) {
				if req.UserID != "U12345" {
					t.Errorf("UserID = %q, want %q", req.UserID, "U12345")
				}
				if req.UserName != "testuser" {
					t.Errorf("UserName = %q, want %q", req.UserName, "testuser")
				}
				if req.TriggerID != "T67890" {
					t.Errorf("TriggerID = %q, want %q", req.TriggerID, "T67890")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto := &SlackCommandDTO{
				Text:      tt.text,
				UserID:    tt.userID,
				UserName:  tt.userName,
				TriggerID: tt.triggerID,
			}

			req := dto.ParseSilenceRequest()

			if req.Action != tt.expectedAction {
				t.Errorf("Action = %v, want %v", req.Action, tt.expectedAction)
			}

			if tt.expectedAction == SilenceActionDelete && req.SilenceID != tt.expectedID {
				t.Errorf("SilenceID = %q, want %q", req.SilenceID, tt.expectedID)
			}

			if tt.validateExtra != nil {
				tt.validateExtra(t, req)
			}

			// Verify matchers map is initialized
			if req.Matchers == nil {
				t.Error("Matchers should not be nil")
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		// Valid durations
		{"1m", 1 * time.Minute},
		{"30m", 30 * time.Minute},
		{"1h", 1 * time.Hour},
		{"24h", 24 * time.Hour},
		{"1d", 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"1w", 7 * 24 * time.Hour},
		{"4w", 28 * 24 * time.Hour},

		// Case insensitivity
		{"1H", 1 * time.Hour},
		{"1D", 24 * time.Hour},
		{"1W", 7 * 24 * time.Hour},
		{"1M", 1 * time.Minute},

		// Invalid durations return 0
		{"", 0},
		{"invalid", 0},
		{"0h", 0},
		{"0m", 0},
		{"-1h", 0},
		{"1.5h", 0},
		{"1x", 0},
		{"h", 0},
		{"1", 0},
		{"1hh", 0},
		{"h1", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseDuration(tt.input); got != tt.expected {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSlackCommandDTO_ParsedArgs(t *testing.T) {
	tests := []struct {
		name             string
		text             string
		expectedSeverity string
	}{
		{"empty text returns empty map", "", ""},
		{"text is set as severity", "critical", "critical"},
		{"warning severity", "warning", "warning"},
		{"info severity", "info", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto := &SlackCommandDTO{Text: tt.text}
			args := dto.ParsedArgs()

			if tt.expectedSeverity == "" {
				if _, exists := args["severity"]; exists {
					t.Error("Expected no severity key in empty text case")
				}
			} else {
				if args["severity"] != tt.expectedSeverity {
					t.Errorf("severity = %q, want %q", args["severity"], tt.expectedSeverity)
				}
			}
		})
	}
}

func TestSlackCommandDTO_SeverityFilter(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"", ""},
		{"critical", "critical"},
		{"warning", "warning"},
		{"info", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			dto := &SlackCommandDTO{Text: tt.text}
			if got := dto.SeverityFilter(); got != tt.expected {
				t.Errorf("SeverityFilter() = %q, want %q", got, tt.expected)
			}
		})
	}
}
