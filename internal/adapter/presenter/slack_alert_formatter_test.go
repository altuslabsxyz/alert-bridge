package presenter

import (
	"strings"
	"testing"
	"time"

	"github.com/altuslabsxyz/alert-bridge/internal/domain/entity"
	slackUseCase "github.com/altuslabsxyz/alert-bridge/internal/usecase/slack"
	"github.com/slack-go/slack"
)

func TestNewSlackAlertFormatter(t *testing.T) {
	f := NewSlackAlertFormatter()
	if f == nil {
		t.Error("NewSlackAlertFormatter() returned nil")
	}
}

func TestSlackAlertFormatter_formatDuration(t *testing.T) {
	f := NewSlackAlertFormatter()

	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			want:     "45s",
		},
		{
			name:     "just under a minute",
			duration: 59 * time.Second,
			want:     "59s",
		},
		{
			name:     "exactly one minute",
			duration: time.Minute,
			want:     "1m",
		},
		{
			name:     "minutes only",
			duration: 30 * time.Minute,
			want:     "30m",
		},
		{
			name:     "just under an hour",
			duration: 59 * time.Minute,
			want:     "59m",
		},
		{
			name:     "exactly one hour",
			duration: time.Hour,
			want:     "1h 0m",
		},
		{
			name:     "hours and minutes",
			duration: 2*time.Hour + 30*time.Minute,
			want:     "2h 30m",
		},
		{
			name:     "just under a day",
			duration: 23*time.Hour + 59*time.Minute,
			want:     "23h 59m",
		},
		{
			name:     "exactly one day",
			duration: 24 * time.Hour,
			want:     "1d 0h",
		},
		{
			name:     "days and hours",
			duration: 3*24*time.Hour + 12*time.Hour,
			want:     "3d 12h",
		},
		{
			name:     "zero duration",
			duration: 0,
			want:     "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestSlackAlertFormatter_getSeverityMarker(t *testing.T) {
	f := NewSlackAlertFormatter()

	tests := []struct {
		name     string
		severity entity.AlertSeverity
		want     string
	}{
		{
			name:     "critical severity",
			severity: entity.SeverityCritical,
			want:     "🔴",
		},
		{
			name:     "warning severity",
			severity: entity.SeverityWarning,
			want:     "🟡",
		},
		{
			name:     "info severity",
			severity: entity.SeverityInfo,
			want:     "🔵",
		},
		{
			name:     "unknown severity",
			severity: entity.AlertSeverity("unknown"),
			want:     "⚪",
		},
		{
			name:     "empty severity",
			severity: entity.AlertSeverity(""),
			want:     "⚪",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.getSeverityMarker(tt.severity)
			if got != tt.want {
				t.Errorf("getSeverityMarker(%q) = %q, want %q", tt.severity, got, tt.want)
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
		{
			name:     "critical",
			severity: "critical",
			want:     "Critical",
		},
		{
			name:     "warning",
			severity: "warning",
			want:     "Warning",
		},
		{
			name:     "info",
			severity: "info",
			want:     "Info",
		},
		{
			name:     "unknown returns as-is",
			severity: "unknown",
			want:     "unknown",
		},
		{
			name:     "empty returns as-is",
			severity: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.formatSeverity(tt.severity)
			if got != tt.want {
				t.Errorf("formatSeverity(%q) = %q, want %q", tt.severity, got, tt.want)
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
		{
			name:    "empty slice",
			details: []string{},
			want:    "",
		},
		{
			name:    "single item",
			details: []string{"one"},
			want:    "one",
		},
		{
			name:    "two items",
			details: []string{"one", "two"},
			want:    "one | two",
		},
		{
			name:    "multiple items",
			details: []string{"a", "b", "c", "d"},
			want:    "a | b | c | d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.joinDetails(tt.details)
			if got != tt.want {
				t.Errorf("joinDetails(%v) = %q, want %q", tt.details, got, tt.want)
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
		wantLines int
		wantFirst string // First line should contain highest count instance
	}{
		{
			name:      "empty map",
			instances: map[string]int{},
			limit:     5,
			wantLines: 0,
			wantFirst: "",
		},
		{
			name:      "single instance",
			instances: map[string]int{"server-1": 5},
			limit:     5,
			wantLines: 1,
			wantFirst: "1. `server-1`: 5 alert(s)",
		},
		{
			name: "sorted by count descending",
			instances: map[string]int{
				"server-low":  2,
				"server-high": 10,
				"server-mid":  5,
			},
			limit:     5,
			wantLines: 3,
			wantFirst: "1. `server-high`: 10 alert(s)",
		},
		{
			name: "respects limit",
			instances: map[string]int{
				"s1": 10,
				"s2": 8,
				"s3": 6,
				"s4": 4,
				"s5": 2,
			},
			limit:     3,
			wantLines: 3,
			wantFirst: "1. `s1`: 10 alert(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.formatTopInstances(tt.instances, tt.limit)

			// Count non-empty lines
			lines := strings.Split(strings.TrimSpace(got), "\n")
			if got == "" {
				lines = []string{}
			}
			if len(lines) != tt.wantLines {
				t.Errorf("formatTopInstances() returned %d lines, want %d", len(lines), tt.wantLines)
			}

			// Check first line contains highest count
			if tt.wantFirst != "" && len(lines) > 0 {
				if !strings.Contains(lines[0], tt.wantFirst) {
					t.Errorf("First line = %q, want to contain %q", lines[0], tt.wantFirst)
				}
			}
		})
	}
}

func TestSlackAlertFormatter_FormatAlertStatus_EmptyAlerts(t *testing.T) {
	f := NewSlackAlertFormatter()
	blocks := f.FormatAlertStatus([]*entity.Alert{}, "")

	// Should have: header, summary, divider, "no alerts" message, footer
	if len(blocks) < 4 {
		t.Errorf("FormatAlertStatus with empty alerts should have at least 4 blocks, got %d", len(blocks))
	}

	// Verify header block exists
	header, ok := blocks[0].(*slack.HeaderBlock)
	if !ok {
		t.Error("First block should be a HeaderBlock")
	} else if header.Text.Text != "Alert Status Dashboard" {
		t.Errorf("Header text = %q, want %q", header.Text.Text, "Alert Status Dashboard")
	}
}

func TestSlackAlertFormatter_FormatAlertStatus_WithSeverityFilter(t *testing.T) {
	f := NewSlackAlertFormatter()
	blocks := f.FormatAlertStatus([]*entity.Alert{}, "critical")

	header, ok := blocks[0].(*slack.HeaderBlock)
	if !ok {
		t.Error("First block should be a HeaderBlock")
		return
	}

	if !strings.Contains(header.Text.Text, "Critical") {
		t.Errorf("Header should contain 'Critical', got %q", header.Text.Text)
	}
}

func TestSlackAlertFormatter_FormatAlertStatus_SingleAlert(t *testing.T) {
	f := NewSlackAlertFormatter()

	alert := createTestAlert("alert-1", "High CPU Usage", entity.SeverityCritical)
	alert.Summary = "CPU usage exceeded 90%"
	alert.Instance = "web-server-01"

	blocks := f.FormatAlertStatus([]*entity.Alert{alert}, "")

	// Verify summary shows correct count
	summaryBlock := findSectionBlock(blocks, "Total Active Alerts: 1")
	if summaryBlock == nil {
		t.Error("Should have a summary block showing 1 alert")
	}
}

func TestSlackAlertFormatter_FormatAlertStatus_TruncatesAt10Alerts(t *testing.T) {
	f := NewSlackAlertFormatter()

	// Create 15 alerts
	alerts := make([]*entity.Alert, 15)
	for i := 0; i < 15; i++ {
		alerts[i] = createTestAlert("alert-"+string(rune('a'+i)), "Alert "+string(rune('A'+i)), entity.SeverityWarning)
	}

	blocks := f.FormatAlertStatus(alerts, "")

	// Should contain truncation message
	found := false
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, "Showing 10 of 15 alerts") {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("Should have truncation message when more than 10 alerts")
	}
}

func TestSlackAlertFormatter_FormatAlertStatus_AcknowledgedAlert(t *testing.T) {
	f := NewSlackAlertFormatter()

	alert := createTestAlert("alert-1", "Disk Space Low", entity.SeverityWarning)
	ackedAt := time.Now().Add(-30 * time.Minute)
	alert.State = entity.StateAcked
	alert.AckedAt = &ackedAt
	alert.AckedBy = "jane.doe"

	blocks := f.FormatAlertStatus([]*entity.Alert{alert}, "")

	// Verify acknowledged state is shown
	found := false
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, "Acknowledged by jane.doe") {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("Acknowledged alert should show acknowledger name")
	}
}

func TestSlackAlertFormatter_FormatAlertSummary_EmptySummary(t *testing.T) {
	f := NewSlackAlertFormatter()
	summary := entity.NewAlertSummary()

	blocks := f.FormatAlertSummary(summary, "last 24 hours")

	// Verify header contains period
	header, ok := blocks[0].(*slack.HeaderBlock)
	if !ok {
		t.Error("First block should be a HeaderBlock")
		return
	}

	if !strings.Contains(header.Text.Text, "last 24 hours") {
		t.Errorf("Header should contain period, got %q", header.Text.Text)
	}

	// Verify severity counts (all zeros)
	foundSeverity := false
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, "Critical: 0") {
				foundSeverity = true
				break
			}
		}
	}

	if !foundSeverity {
		t.Error("Should show severity breakdown with zero counts")
	}
}

func TestSlackAlertFormatter_FormatAlertSummary_WithData(t *testing.T) {
	f := NewSlackAlertFormatter()
	summary := &entity.AlertSummary{
		TotalAlerts: 15,
		AlertsBySeverity: map[entity.AlertSeverity]int{
			entity.SeverityCritical: 3,
			entity.SeverityWarning:  7,
			entity.SeverityInfo:     5,
		},
		AlertsByState: map[entity.AlertState]int{
			entity.StateActive: 10,
			entity.StateAcked:  5,
		},
		AlertsByInstance: map[string]int{
			"server-1": 8,
			"server-2": 4,
			"server-3": 3,
		},
		TopAcknowledgers: []entity.UserAckCount{
			{UserName: "alice", Count: 5},
			{UserName: "bob", Count: 3},
		},
	}

	blocks := f.FormatAlertSummary(summary, "last 7 days")

	// Verify total alerts
	foundTotal := false
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, "Total Alerts:* 15") {
				foundTotal = true
				break
			}
		}
	}

	if !foundTotal {
		t.Error("Should show total alerts count")
	}

	// Verify top acknowledgers
	foundAckers := false
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, "alice") {
				foundAckers = true
				break
			}
		}
	}

	if !foundAckers {
		t.Error("Should show top acknowledgers")
	}
}

func TestSlackAlertFormatter_FormatAlertSummary_NoAcknowledgers(t *testing.T) {
	f := NewSlackAlertFormatter()
	summary := &entity.AlertSummary{
		TotalAlerts:      5,
		AlertsBySeverity: map[entity.AlertSeverity]int{},
		AlertsByState:    map[entity.AlertState]int{},
		AlertsByInstance: map[string]int{},
		TopAcknowledgers: []entity.UserAckCount{},
	}

	blocks := f.FormatAlertSummary(summary, "")

	// Should show "No acknowledgments recorded"
	found := false
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, "No acknowledgments recorded") {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("Should show 'No acknowledgments recorded' when no acknowledgers")
	}
}

func TestSlackAlertFormatter_FormatAlertSummary_AllTimePeriod(t *testing.T) {
	f := NewSlackAlertFormatter()
	summary := entity.NewAlertSummary()

	blocks := f.FormatAlertSummary(summary, "all time")

	header, ok := blocks[0].(*slack.HeaderBlock)
	if !ok {
		t.Error("First block should be a HeaderBlock")
		return
	}

	// "all time" should not be included in header
	if strings.Contains(header.Text.Text, "all time") {
		t.Errorf("Header should not contain 'all time', got %q", header.Text.Text)
	}
}

func TestSlackAlertFormatter_FormatSilenceResult_Created(t *testing.T) {
	f := NewSlackAlertFormatter()

	silence := createTestSilence("silence-1", 2*time.Hour, "admin")
	result := &slackUseCase.SilenceResult{
		Created: silence,
		Message: "Silence created successfully",
	}

	blocks := f.FormatSilenceResult(result)

	// Verify header
	header, ok := blocks[0].(*slack.HeaderBlock)
	if !ok {
		t.Error("First block should be a HeaderBlock")
		return
	}

	if header.Text.Text != "Silence Management" {
		t.Errorf("Header = %q, want %q", header.Text.Text, "Silence Management")
	}

	// Verify message is shown
	found := false
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, "Silence created successfully") {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("Should show result message")
	}

	// Verify created silence details are shown
	foundCreated := false
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, "Created Silence") {
				foundCreated = true
				break
			}
		}
	}

	if !foundCreated {
		t.Error("Should show 'Created Silence' prefix")
	}
}

func TestSlackAlertFormatter_FormatSilenceResult_Deleted(t *testing.T) {
	f := NewSlackAlertFormatter()

	silence := createTestSilence("silence-1", 0, "admin")
	result := &slackUseCase.SilenceResult{
		Deleted: silence,
		Message: "Silence deleted",
	}

	blocks := f.FormatSilenceResult(result)

	// Verify deleted silence details are shown
	foundDeleted := false
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, "Deleted Silence") {
				foundDeleted = true
				break
			}
		}
	}

	if !foundDeleted {
		t.Error("Should show 'Deleted Silence' prefix")
	}
}

func TestSlackAlertFormatter_FormatSilenceResult_ListSilences(t *testing.T) {
	f := NewSlackAlertFormatter()

	silences := []*entity.SilenceMark{
		createTestSilence("s1", time.Hour, "user1"),
		createTestSilence("s2", 2*time.Hour, "user2"),
		createTestSilence("s3", 3*time.Hour, "user3"),
	}

	result := &slackUseCase.SilenceResult{
		Silences: silences,
		Message:  "Active silences",
	}

	blocks := f.FormatSilenceResult(result)

	// Verify all silences are shown
	silenceCount := 0
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, "ID: `") {
				silenceCount++
			}
		}
	}

	if silenceCount != 3 {
		t.Errorf("Should show 3 silence details, got %d", silenceCount)
	}
}

func TestSlackAlertFormatter_FormatSilenceResult_TruncatesAt10(t *testing.T) {
	f := NewSlackAlertFormatter()

	// Create 15 silences
	silences := make([]*entity.SilenceMark, 15)
	for i := 0; i < 15; i++ {
		silences[i] = createTestSilence("silence-"+string(rune('a'+i)), time.Hour, "user")
	}

	result := &slackUseCase.SilenceResult{
		Silences: silences,
		Message:  "Active silences",
	}

	blocks := f.FormatSilenceResult(result)

	// Should contain truncation message
	found := false
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, "Showing 10 of 15 silences") {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("Should have truncation message when more than 10 silences")
	}
}

func TestSlackAlertFormatter_FormatSilenceResult_NoSilences(t *testing.T) {
	f := NewSlackAlertFormatter()

	result := &slackUseCase.SilenceResult{
		Silences: []*entity.SilenceMark{},
		Message:  "No active silences",
	}

	blocks := f.FormatSilenceResult(result)

	// Should show "No active silences" message
	found := false
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, "No active silences") {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("Should show 'No active silences' when silence list is empty")
	}
}

func TestSlackAlertFormatter_FormatSilenceResult_WithReason(t *testing.T) {
	f := NewSlackAlertFormatter()

	silence := createTestSilence("silence-1", 2*time.Hour, "admin")
	silence.Reason = "Maintenance window"

	result := &slackUseCase.SilenceResult{
		Created: silence,
		Message: "Silence created",
	}

	blocks := f.FormatSilenceResult(result)

	// Verify reason is shown
	found := false
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, "Maintenance window") {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("Should show silence reason")
	}
}

// Helper functions

func createTestAlert(id, name string, severity entity.AlertSeverity) *entity.Alert {
	return &entity.Alert{
		ID:        id,
		Name:      name,
		Severity:  severity,
		State:     entity.StateActive,
		FiredAt:   time.Now().Add(-30 * time.Minute),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func createTestSilence(id string, remaining time.Duration, createdBy string) *entity.SilenceMark {
	now := time.Now()
	return &entity.SilenceMark{
		ID:        id,
		StartAt:   now.Add(-time.Hour),
		EndAt:     now.Add(remaining),
		CreatedBy: createdBy,
		CreatedAt: now.Add(-time.Hour),
	}
}

func findSectionBlock(blocks []slack.Block, containsText string) *slack.SectionBlock {
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil && strings.Contains(section.Text.Text, containsText) {
				return section
			}
		}
	}
	return nil
}
