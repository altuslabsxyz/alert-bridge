package slack

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/slack-go/slack"

	domainerrors "github.com/altuslabsxyz/alert-bridge/internal/domain/errors"
)

func TestParseMessageID(t *testing.T) {
	tests := []struct {
		name          string
		messageID     string
		wantChannelID string
		wantTimestamp string
		wantErr       bool
	}{
		{
			name:          "valid message ID",
			messageID:     "C12345:1234567890.123456",
			wantChannelID: "C12345",
			wantTimestamp: "1234567890.123456",
			wantErr:       false,
		},
		{
			name:          "valid with longer channel ID",
			messageID:     "C0123456789:1609459200.000100",
			wantChannelID: "C0123456789",
			wantTimestamp: "1609459200.000100",
			wantErr:       false,
		},
		{
			name:          "timestamp with colons preserved",
			messageID:     "C123:ts:with:colons",
			wantChannelID: "C123",
			wantTimestamp: "ts:with:colons",
			wantErr:       false,
		},
		{
			name:      "no colon separator",
			messageID: "invalid-message-id",
			wantErr:   true,
		},
		{
			name:      "empty string",
			messageID: "",
			wantErr:   true,
		},
		{
			name:      "only colon",
			messageID: ":",
			wantErr:   false, // splits to ["", ""] which has len 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channelID, timestamp, err := parseMessageID(tt.messageID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseMessageID() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseMessageID() unexpected error: %v", err)
				return
			}

			if channelID != tt.wantChannelID {
				t.Errorf("channelID = %q, want %q", channelID, tt.wantChannelID)
			}
			if timestamp != tt.wantTimestamp {
				t.Errorf("timestamp = %q, want %q", timestamp, tt.wantTimestamp)
			}
		})
	}
}

// mockNetError implements net.Error for testing
type mockNetError struct {
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return "mock network error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

func TestCategorizeSlackError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		operation    string
		wantCategory domainerrors.ErrorCategory
		wantNil      bool
	}{
		{
			name:      "nil error returns nil",
			err:       nil,
			operation: "test",
			wantNil:   true,
		},
		{
			name:         "network error is transient",
			err:          &mockNetError{timeout: true},
			operation:    "posting message",
			wantCategory: domainerrors.CategoryTransient,
		},
		{
			name:         "context deadline exceeded is transient",
			err:          context.DeadlineExceeded,
			operation:    "api call",
			wantCategory: domainerrors.CategoryTransient,
		},
		{
			name:         "context canceled is transient",
			err:          context.Canceled,
			operation:    "api call",
			wantCategory: domainerrors.CategoryTransient,
		},
		{
			name:         "rate_limited slack error is transient",
			err:          slack.SlackErrorResponse{Err: "rate_limited"},
			operation:    "posting",
			wantCategory: domainerrors.CategoryTransient,
		},
		{
			name:         "internal_error slack error is transient",
			err:          slack.SlackErrorResponse{Err: "internal_error"},
			operation:    "posting",
			wantCategory: domainerrors.CategoryTransient,
		},
		{
			name:         "fatal_error slack error is transient",
			err:          slack.SlackErrorResponse{Err: "fatal_error"},
			operation:    "posting",
			wantCategory: domainerrors.CategoryTransient,
		},
		{
			name:         "service_unavailable slack error is transient",
			err:          slack.SlackErrorResponse{Err: "service_unavailable"},
			operation:    "posting",
			wantCategory: domainerrors.CategoryTransient,
		},
		{
			name:         "invalid_auth slack error is permanent",
			err:          slack.SlackErrorResponse{Err: "invalid_auth"},
			operation:    "auth",
			wantCategory: domainerrors.CategoryPermanent,
		},
		{
			name:         "account_inactive slack error is permanent",
			err:          slack.SlackErrorResponse{Err: "account_inactive"},
			operation:    "posting",
			wantCategory: domainerrors.CategoryPermanent,
		},
		{
			name:         "token_revoked slack error is permanent",
			err:          slack.SlackErrorResponse{Err: "token_revoked"},
			operation:    "posting",
			wantCategory: domainerrors.CategoryPermanent,
		},
		{
			name:         "no_permission slack error is permanent",
			err:          slack.SlackErrorResponse{Err: "no_permission"},
			operation:    "posting",
			wantCategory: domainerrors.CategoryPermanent,
		},
		{
			name:         "channel_not_found slack error is permanent",
			err:          slack.SlackErrorResponse{Err: "channel_not_found"},
			operation:    "posting",
			wantCategory: domainerrors.CategoryPermanent,
		},
		{
			name:         "not_in_channel slack error is permanent",
			err:          slack.SlackErrorResponse{Err: "not_in_channel"},
			operation:    "posting",
			wantCategory: domainerrors.CategoryPermanent,
		},
		{
			name:         "is_archived slack error is permanent",
			err:          slack.SlackErrorResponse{Err: "is_archived"},
			operation:    "posting",
			wantCategory: domainerrors.CategoryPermanent,
		},
		{
			name:         "unknown slack error defaults to permanent",
			err:          slack.SlackErrorResponse{Err: "some_unknown_error"},
			operation:    "posting",
			wantCategory: domainerrors.CategoryPermanent,
		},
		{
			name:         "generic error defaults to permanent",
			err:          errors.New("unknown error"),
			operation:    "posting",
			wantCategory: domainerrors.CategoryPermanent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeSlackError(tt.err, tt.operation)

			if tt.wantNil {
				if result != nil {
					t.Errorf("categorizeSlackError() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("categorizeSlackError() = nil, want non-nil")
			}

			// Check that it's a DomainError
			var domainErr *domainerrors.DomainError
			if !errors.As(result, &domainErr) {
				t.Fatalf("result is not a DomainError: %T", result)
			}

			if domainErr.Category != tt.wantCategory {
				t.Errorf("error category = %v, want %v", domainErr.Category, tt.wantCategory)
			}
		})
	}
}

func TestCategorizeSlackError_WrappedNetError(t *testing.T) {
	// Test that wrapped network errors are still detected
	wrappedErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &mockNetError{timeout: true},
	}

	result := categorizeSlackError(wrappedErr, "connecting")
	if result == nil {
		t.Fatal("expected non-nil error")
	}

	var domainErr *domainerrors.DomainError
	if !errors.As(result, &domainErr) {
		t.Fatalf("result is not a DomainError: %T", result)
	}

	if domainErr.Category != domainerrors.CategoryTransient {
		t.Errorf("wrapped net error should be transient, got %v", domainErr.Category)
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name             string
		botToken         string
		channelID        string
		silenceDurations []int // minutes
		apiURL           []string
	}{
		{
			name:             "basic client creation",
			botToken:         "xoxb-test-token",
			channelID:        "C12345",
			silenceDurations: nil,
		},
		{
			name:             "with custom silence durations",
			botToken:         "xoxb-test-token",
			channelID:        "C12345",
			silenceDurations: []int{15, 60},
		},
		{
			name:             "with custom API URL",
			botToken:         "xoxb-test-token",
			channelID:        "C12345",
			silenceDurations: nil,
			apiURL:           []string{"https://custom-api.example.com/api/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var durations []time.Duration
			for _, m := range tt.silenceDurations {
				durations = append(durations, time.Duration(m)*time.Minute)
			}

			client := NewClient(tt.botToken, tt.channelID, durations, tt.apiURL...)

			if client == nil {
				t.Fatal("NewClient() returned nil")
			}
			if client.channelID != tt.channelID {
				t.Errorf("channelID = %q, want %q", client.channelID, tt.channelID)
			}
			if client.api == nil {
				t.Error("api client is nil")
			}
			if client.messageBuilder == nil {
				t.Error("messageBuilder is nil")
			}
		})
	}
}

func TestClient_Name(t *testing.T) {
	client := NewClient("token", "channel", nil)
	if client.Name() != "slack" {
		t.Errorf("Name() = %q, want %q", client.Name(), "slack")
	}
}
