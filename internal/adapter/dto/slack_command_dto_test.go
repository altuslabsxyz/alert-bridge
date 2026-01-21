package dto

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Duration
	}{
		{"minutes", "30m", 30 * time.Minute},
		{"hours", "2h", 2 * time.Hour},
		{"days", "7d", 7 * 24 * time.Hour},
		{"weeks", "1w", 7 * 24 * time.Hour},
		{"invalid", "abc", 0},
		{"empty", "", 0},
		{"no_unit", "123", 0},
		{"negative", "-1h", 0},
		{"zero", "0h", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDuration(tt.input)
			if got != tt.want {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSlackCommandDTO_PeriodFilter(t *testing.T) {
	tests := []struct {
		name string
		text string
		want time.Duration
	}{
		{"empty", "", 0},
		{"today", "today", 24 * time.Hour},
		{"week", "week", 7 * 24 * time.Hour},
		{"thisweek", "thisweek", 7 * 24 * time.Hour},
		{"all", "all", 0},
		{"1h", "1h", time.Hour},
		{"24h", "24h", 24 * time.Hour},
		{"7d", "7d", 7 * 24 * time.Hour},
		{"invalid", "invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto := &SlackCommandDTO{Text: tt.text}
			got := dto.PeriodFilter()
			if got != tt.want {
				t.Errorf("PeriodFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackCommandDTO_PeriodDescription(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"empty", "", "all time"},
		{"all", "all", "all time"},
		{"1h", "1h", "last 1 hour(s)"},
		{"24h", "24h", "last 1 day(s)"},
		{"7d", "7d", "last 1 week(s)"},
		{"3d", "3d", "last 3 day(s)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto := &SlackCommandDTO{Text: tt.text}
			got := dto.PeriodDescription()
			if got != tt.want {
				t.Errorf("PeriodDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackCommandDTO_SeverityFilter(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"empty", "", ""},
		{"critical", "critical", "critical"},
		{"warning", "warning", "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto := &SlackCommandDTO{Text: tt.text}
			got := dto.SeverityFilter()
			if got != tt.want {
				t.Errorf("SeverityFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackCommandDTO_ParseSilenceRequest(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantAction SilenceAction
	}{
		{"empty defaults to list", "", SilenceActionList},
		{"list", "list", SilenceActionList},
		{"create", "create", SilenceActionOpenModal},
		{"delete", "delete abc123", SilenceActionDelete},
		{"duration opens modal", "1h", SilenceActionOpenModal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto := &SlackCommandDTO{
				Text:     tt.text,
				UserID:   "U123",
				UserName: "testuser",
			}
			got := dto.ParseSilenceRequest()
			if got.Action != tt.wantAction {
				t.Errorf("ParseSilenceRequest().Action = %v, want %v", got.Action, tt.wantAction)
			}
		})
	}

	t.Run("delete with ID", func(t *testing.T) {
		dto := &SlackCommandDTO{Text: "delete silence-123"}
		req := dto.ParseSilenceRequest()
		if req.SilenceID != "silence-123" {
			t.Errorf("SilenceID = %v, want %v", req.SilenceID, "silence-123")
		}
	})
}
