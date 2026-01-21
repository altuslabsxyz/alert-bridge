package slack

import (
	"testing"
	"time"

	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/entity"
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

func TestBuildUserMentions(t *testing.T) {
	tests := []struct {
		name    string
		userIDs []string
		want    string
	}{
		{
			name:    "empty user list",
			userIDs: []string{},
			want:    "",
		},
		{
			name:    "single user",
			userIDs: []string{"U12345"},
			want:    "<@U12345>",
		},
		{
			name:    "multiple users",
			userIDs: []string{"U12345", "U67890", "UABCDE"},
			want:    "<@U12345> <@U67890> <@UABCDE>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildUserMentions(tt.userIDs)
			if got != tt.want {
				t.Errorf("BuildUserMentions() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildAlertMessageWithMentions(t *testing.T) {
	builder := NewMessageBuilder(nil)

	t.Run("message with mentions includes subscriber section", func(t *testing.T) {
		alert := createTestAlert()
		slackUserIDs := []string{"U12345", "U67890"}

		blocks := builder.BuildAlertMessageWithMentions(alert, slackUserIDs)

		// First block should be the mentions section
		if len(blocks) == 0 {
			t.Fatal("expected at least one block")
		}

		// Verify mentions section exists by checking block count is greater than without mentions
		blocksWithoutMentions := builder.BuildAlertMessage(alert)
		if len(blocks) <= len(blocksWithoutMentions) {
			t.Errorf("expected more blocks with mentions (%d) than without (%d)", len(blocks), len(blocksWithoutMentions))
		}
	})

	t.Run("message without mentions has no subscriber section", func(t *testing.T) {
		alert := createTestAlert()

		blocksWithEmpty := builder.BuildAlertMessageWithMentions(alert, []string{})
		blocksWithNil := builder.BuildAlertMessageWithMentions(alert, nil)
		blocksNormal := builder.BuildAlertMessage(alert)

		// All should have the same number of blocks
		if len(blocksWithEmpty) != len(blocksNormal) {
			t.Errorf("empty mentions should produce same blocks as normal: got %d, want %d", len(blocksWithEmpty), len(blocksNormal))
		}
		if len(blocksWithNil) != len(blocksNormal) {
			t.Errorf("nil mentions should produce same blocks as normal: got %d, want %d", len(blocksWithNil), len(blocksNormal))
		}
	})
}

func createTestAlert() *entity.Alert {
	alert := entity.NewAlert(
		"fingerprint123",
		"TestAlert",
		"instance-1",
		"target-1",
		"Test alert summary",
		entity.SeverityWarning,
	)
	alert.FiredAt = time.Date(2024, 1, 21, 15, 0, 0, 0, time.UTC)
	return alert
}
