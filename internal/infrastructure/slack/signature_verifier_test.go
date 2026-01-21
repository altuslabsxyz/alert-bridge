package slack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestSignatureVerifier_VerifySignature(t *testing.T) {
	secret := "test-signing-secret-12345"
	verifier := NewSignatureVerifier(secret)

	// Helper to generate valid signature
	generateSignature := func(timestamp string, body []byte, signingSecret string) string {
		baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
		h := hmac.New(sha256.New, []byte(signingSecret))
		h.Write([]byte(baseString))
		return "v0=" + hex.EncodeToString(h.Sum(nil))
	}

	tests := []struct {
		name        string
		timestamp   string
		body        []byte
		signature   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid signature",
			timestamp: strconv.FormatInt(time.Now().Unix(), 10),
			body:      []byte(`{"type":"event_callback","event":{"type":"message"}}`),
			signature: "", // will be generated
			wantErr:   false,
		},
		{
			name:      "valid signature with empty body",
			timestamp: strconv.FormatInt(time.Now().Unix(), 10),
			body:      []byte{},
			signature: "", // will be generated
			wantErr:   false,
		},
		{
			name:        "invalid signature format - no prefix",
			timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
			body:        []byte(`{"test":"data"}`),
			signature:   "invalid-signature-without-prefix",
			wantErr:     true,
			errContains: "invalid signature format",
		},
		{
			name:        "invalid signature format - wrong version",
			timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
			body:        []byte(`{"test":"data"}`),
			signature:   "v1=abcdef1234567890",
			wantErr:     true,
			errContains: "invalid signature format",
		},
		{
			name:        "signature mismatch - wrong secret",
			timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
			body:        []byte(`{"test":"data"}`),
			signature:   "WRONG_SECRET", // will generate with wrong secret
			wantErr:     true,
			errContains: "signature mismatch",
		},
		{
			name:        "signature mismatch - tampered body",
			timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
			body:        []byte(`{"test":"tampered"}`),
			signature:   "TAMPERED_BODY", // will generate with original body
			wantErr:     true,
			errContains: "signature mismatch",
		},
		{
			name:        "timestamp too old",
			timestamp:   strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10),
			body:        []byte(`{"test":"data"}`),
			signature:   "", // will be generated
			wantErr:     true,
			errContains: "timestamp too old",
		},
		{
			name:        "timestamp in future",
			timestamp:   strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10),
			body:        []byte(`{"test":"data"}`),
			signature:   "", // will be generated
			wantErr:     true,
			errContains: "timestamp is in the future",
		},
		{
			name:        "invalid timestamp format",
			timestamp:   "not-a-number",
			body:        []byte(`{"test":"data"}`),
			signature:   "v0=abcdef",
			wantErr:     true,
			errContains: "invalid timestamp format",
		},
		{
			name:        "empty timestamp",
			timestamp:   "",
			body:        []byte(`{"test":"data"}`),
			signature:   "v0=abcdef",
			wantErr:     true,
			errContains: "invalid timestamp format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signature := tt.signature

			// Generate valid signature for certain test cases
			if signature == "" {
				signature = generateSignature(tt.timestamp, tt.body, secret)
			} else if signature == "WRONG_SECRET" {
				signature = generateSignature(tt.timestamp, tt.body, "wrong-secret")
			} else if signature == "TAMPERED_BODY" {
				signature = generateSignature(tt.timestamp, []byte(`{"test":"original"}`), secret)
			}

			err := verifier.VerifySignature(tt.timestamp, tt.body, signature)

			if tt.wantErr {
				if err == nil {
					t.Errorf("VerifySignature() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("VerifySignature() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("VerifySignature() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestSignatureVerifier_TimestampBoundary(t *testing.T) {
	secret := "test-secret"
	verifier := NewSignatureVerifier(secret)

	generateSignature := func(timestamp string, body []byte) string {
		baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
		h := hmac.New(sha256.New, []byte(secret))
		h.Write([]byte(baseString))
		return "v0=" + hex.EncodeToString(h.Sum(nil))
	}

	body := []byte(`{"test":"boundary"}`)

	tests := []struct {
		name      string
		offset    time.Duration
		wantErr   bool
		errPrefix string
	}{
		{
			name:    "exactly at current time",
			offset:  0,
			wantErr: false,
		},
		{
			name:    "4 minutes ago (within limit)",
			offset:  -4 * time.Minute,
			wantErr: false,
		},
		{
			name:      "6 minutes ago (exceeds 5 min limit)",
			offset:    -6 * time.Minute,
			wantErr:   true,
			errPrefix: "timestamp too old",
		},
		{
			name:    "30 seconds in future (within 1 min tolerance)",
			offset:  30 * time.Second,
			wantErr: false,
		},
		{
			name:      "2 minutes in future (exceeds 1 min tolerance)",
			offset:    2 * time.Minute,
			wantErr:   true,
			errPrefix: "timestamp is in the future",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timestamp := strconv.FormatInt(time.Now().Add(tt.offset).Unix(), 10)
			signature := generateSignature(timestamp, body)

			err := verifier.VerifySignature(timestamp, body, signature)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if tt.errPrefix != "" && !containsString(err.Error(), tt.errPrefix) {
					t.Errorf("error = %v, want prefix %q", err, tt.errPrefix)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error = %v", err)
				}
			}
		})
	}
}

func TestSignatureVerifier_ComputeSignature(t *testing.T) {
	// Test that computeSignature produces correct HMAC-SHA256
	verifier := NewSignatureVerifier("test-secret")

	baseString := "v0:1234567890:{\"test\":\"data\"}"
	got := verifier.computeSignature(baseString)

	// Independently compute expected value
	h := hmac.New(sha256.New, []byte("test-secret"))
	h.Write([]byte(baseString))
	expected := hex.EncodeToString(h.Sum(nil))

	if got != expected {
		t.Errorf("computeSignature() = %v, want %v", got, expected)
	}
}

func TestNewSignatureVerifier(t *testing.T) {
	verifier := NewSignatureVerifier("my-secret")
	if verifier == nil {
		t.Error("NewSignatureVerifier() returned nil")
	}
	if verifier.signingSecret != "my-secret" {
		t.Errorf("signingSecret = %q, want %q", verifier.signingSecret, "my-secret")
	}
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
