package slack

import (
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(5)

	if cb == nil {
		t.Fatal("NewCircuitBreaker() returned nil")
	}
	if cb.maxFailures != 5 {
		t.Errorf("maxFailures = %d, want 5", cb.maxFailures)
	}
	if cb.consecutiveFailures != 0 {
		t.Errorf("consecutiveFailures = %d, want 0", cb.consecutiveFailures)
	}
	if cb.isOpen {
		t.Error("circuit breaker should start closed")
	}
}

func TestCircuitBreaker_RecordFailure(t *testing.T) {
	cb := NewCircuitBreaker(3)

	// First two failures should not open the circuit
	for i := 1; i <= 2; i++ {
		opened := cb.RecordFailure()
		if opened {
			t.Errorf("RecordFailure() = true on failure %d, want false", i)
		}
		if cb.IsOpen() {
			t.Errorf("circuit should be closed after %d failures", i)
		}
		if cb.ConsecutiveFailures() != i {
			t.Errorf("ConsecutiveFailures() = %d, want %d", cb.ConsecutiveFailures(), i)
		}
	}

	// Third failure should open the circuit
	opened := cb.RecordFailure()
	if !opened {
		t.Error("RecordFailure() = false on 3rd failure, want true")
	}
	if !cb.IsOpen() {
		t.Error("circuit should be open after 3 failures")
	}
	if cb.ConsecutiveFailures() != 3 {
		t.Errorf("ConsecutiveFailures() = %d, want 3", cb.ConsecutiveFailures())
	}
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker(2)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	if !cb.IsOpen() {
		t.Fatal("circuit should be open after 2 failures")
	}

	// Success should reset the circuit
	cb.RecordSuccess()

	if cb.IsOpen() {
		t.Error("circuit should be closed after success")
	}
	if cb.ConsecutiveFailures() != 0 {
		t.Errorf("ConsecutiveFailures() = %d, want 0 after success", cb.ConsecutiveFailures())
	}
}

func TestCircuitBreaker_LastFailureTimestamp(t *testing.T) {
	cb := NewCircuitBreaker(5)

	before := time.Now()
	cb.RecordFailure()
	after := time.Now()

	if cb.lastFailure.Before(before) || cb.lastFailure.After(after) {
		t.Errorf("lastFailure = %v, want between %v and %v", cb.lastFailure, before, after)
	}
}

func TestShouldRetry(t *testing.T) {
	cb := NewCircuitBreaker(2)

	// Should retry when circuit is closed
	if !ShouldRetry(cb) {
		t.Error("ShouldRetry() = false when circuit closed, want true")
	}

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Should not retry when circuit is open
	if ShouldRetry(cb) {
		t.Error("ShouldRetry() = true when circuit open, want false")
	}

	// Reset and try again
	cb.RecordSuccess()
	if !ShouldRetry(cb) {
		t.Error("ShouldRetry() = false after reset, want true")
	}
}

func TestDefaultReconnectionConfig(t *testing.T) {
	cfg := DefaultReconnectionConfig()

	if cfg.InitialBackoff != 500*time.Millisecond {
		t.Errorf("InitialBackoff = %v, want 500ms", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 60*time.Second {
		t.Errorf("MaxBackoff = %v, want 60s", cfg.MaxBackoff)
	}
	if cfg.BackoffMultiplier != 1.5 {
		t.Errorf("BackoffMultiplier = %v, want 1.5", cfg.BackoffMultiplier)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}
}

func TestCalculateBackoff(t *testing.T) {
	cfg := ReconnectionConfig{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond}, // 100ms * 2^0 = 100ms
		{1, 200 * time.Millisecond}, // 100ms * 2^1 = 200ms
		{2, 400 * time.Millisecond}, // 100ms * 2^2 = 400ms
		{3, 800 * time.Millisecond}, // 100ms * 2^3 = 800ms
		{4, 1 * time.Second},        // 100ms * 2^4 = 1600ms, capped at 1s
		{5, 1 * time.Second},        // capped at 1s
		{10, 1 * time.Second},       // capped at 1s
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := CalculateBackoff(cfg, tt.attempt)
			if got != tt.expected {
				t.Errorf("CalculateBackoff(attempt=%d) = %v, want %v", tt.attempt, got, tt.expected)
			}
		})
	}
}

func TestCalculateBackoff_DefaultConfig(t *testing.T) {
	cfg := DefaultReconnectionConfig()

	// Test exponential growth with default config
	tests := []struct {
		attempt     int
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{0, 500 * time.Millisecond, 500 * time.Millisecond},
		{1, 750 * time.Millisecond, 750 * time.Millisecond},   // 500 * 1.5
		{2, 1125 * time.Millisecond, 1125 * time.Millisecond}, // 500 * 1.5^2
	}

	for _, tt := range tests {
		got := CalculateBackoff(cfg, tt.attempt)
		if got < tt.expectedMin || got > tt.expectedMax {
			t.Errorf("CalculateBackoff(attempt=%d) = %v, want between %v and %v",
				tt.attempt, got, tt.expectedMin, tt.expectedMax)
		}
	}

	// Verify it caps at MaxBackoff for large attempts
	got := CalculateBackoff(cfg, 100)
	if got != cfg.MaxBackoff {
		t.Errorf("CalculateBackoff(attempt=100) = %v, want max backoff %v", got, cfg.MaxBackoff)
	}
}

func TestCircuitBreaker_PartialRecovery(t *testing.T) {
	// Test that partial failures followed by success resets properly
	cb := NewCircuitBreaker(5)

	// 3 failures
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.ConsecutiveFailures() != 3 {
		t.Fatalf("expected 3 failures, got %d", cb.ConsecutiveFailures())
	}

	// Success resets
	cb.RecordSuccess()

	if cb.ConsecutiveFailures() != 0 {
		t.Errorf("failures should reset to 0, got %d", cb.ConsecutiveFailures())
	}

	// Start failing again - should count from 0
	cb.RecordFailure()
	if cb.ConsecutiveFailures() != 1 {
		t.Errorf("expected 1 failure after reset, got %d", cb.ConsecutiveFailures())
	}
}

func TestCalculateBackoff_ZeroAttempt(t *testing.T) {
	cfg := ReconnectionConfig{
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
	}

	got := CalculateBackoff(cfg, 0)
	if got != 1*time.Second {
		t.Errorf("CalculateBackoff(0) = %v, want %v", got, 1*time.Second)
	}
}

func TestCalculateBackoff_SmallMultiplier(t *testing.T) {
	cfg := ReconnectionConfig{
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 1.1, // Small multiplier
	}

	prev := CalculateBackoff(cfg, 0)
	for i := 1; i <= 5; i++ {
		curr := CalculateBackoff(cfg, i)
		if curr < prev {
			t.Errorf("backoff should not decrease: attempt %d = %v, previous = %v", i, curr, prev)
		}
		prev = curr
	}
}
