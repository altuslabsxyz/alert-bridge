package slack

import (
	"testing"
	"time"
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
