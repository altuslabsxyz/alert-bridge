// Package harness provides an in-process test harness for e2e testing.
// It creates a fully-functional Alert-Bridge server with mock notifiers.
package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/qj0r9j0vc2/alert-bridge/internal/adapter/dto"
	"github.com/qj0r9j0vc2/alert-bridge/internal/adapter/handler"
	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/repository"
	"github.com/qj0r9j0vc2/alert-bridge/internal/infrastructure/persistence/memory"
	"github.com/qj0r9j0vc2/alert-bridge/internal/usecase/alert"
	"github.com/qj0r9j0vc2/alert-bridge/test/e2e/mocks"
)

// TestHarness manages the in-process test environment
type TestHarness struct {
	t *testing.T

	// Mock notifiers
	SlackNotifier     *mocks.MockNotifier
	PagerDutyNotifier *mocks.MockNotifier

	// Repositories
	AlertRepo   repository.AlertRepository
	AckRepo     repository.AckEventRepository
	SilenceRepo repository.SilenceRepository

	// Use cases
	ProcessAlertUseCase *alert.ProcessAlertUseCase

	// HTTP handler
	AlertmanagerHandler *handler.AlertmanagerHandler

	// Server
	Server *httptest.Server

	// Logger
	Logger *slog.Logger
}

// NewTestHarness creates a new test harness with all components wired together
func NewTestHarness(t *testing.T) *TestHarness {
	t.Helper()

	h := &TestHarness{
		t: t,
	}

	// Create logger (suppress during tests unless verbose)
	h.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Create mock notifiers
	h.SlackNotifier = mocks.NewMockNotifier("slack")
	h.PagerDutyNotifier = mocks.NewMockNotifier("pagerduty")

	// Create in-memory repositories
	h.AlertRepo = memory.NewAlertRepository()
	h.AckRepo = memory.NewAckEventRepository()
	h.SilenceRepo = memory.NewSilenceRepository()

	// Create use case with mock notifiers
	notifiers := []alert.Notifier{
		h.SlackNotifier,
		h.PagerDutyNotifier,
	}

	h.ProcessAlertUseCase = alert.NewProcessAlertUseCase(
		h.AlertRepo,
		h.SilenceRepo,
		notifiers,
		h.Logger,
		nil, // No metrics for testing
	)

	// Create HTTP handler
	h.AlertmanagerHandler = handler.NewAlertmanagerHandler(h.ProcessAlertUseCase, h.Logger)

	// Create test server with mux
	mux := http.NewServeMux()
	mux.Handle("/webhook/alertmanager", h.AlertmanagerHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	h.Server = httptest.NewServer(mux)

	t.Cleanup(func() {
		h.Server.Close()
	})

	return h
}

// Reset clears all state in the harness
func (h *TestHarness) Reset() {
	h.SlackNotifier.Reset()
	h.PagerDutyNotifier.Reset()
	// Note: memory repositories don't have a Reset method, but they're recreated for each test
}

// ServerURL returns the base URL of the test server
func (h *TestHarness) ServerURL() string {
	return h.Server.URL
}

// WebhookURL returns the full webhook URL
func (h *TestHarness) WebhookURL() string {
	return h.Server.URL + "/webhook/alertmanager"
}

// SendAlert sends an alert to the test server
func (h *TestHarness) SendAlert(alerts []dto.AlertmanagerAlert) (*http.Response, error) {
	// Determine webhook status based on alerts
	webhookStatus := "firing"
	allResolved := true
	for _, alert := range alerts {
		if alert.Status != "resolved" {
			allResolved = false
			break
		}
	}
	if allResolved && len(alerts) > 0 {
		webhookStatus = "resolved"
	}

	payload := dto.AlertmanagerWebhook{
		Version:         "4",
		GroupKey:        "test-group",
		TruncatedAlerts: 0,
		Status:          webhookStatus,
		Receiver:        "alert-bridge",
		GroupLabels:     make(map[string]string),
		CommonLabels:    make(map[string]string),
		Alerts:          alerts,
		ExternalURL:     "http://alertmanager:9093",
	}

	if len(alerts) > 0 {
		payload.CommonLabels = alerts[0].Labels
		payload.CommonAnnotations = alerts[0].Annotations
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", h.WebhookURL(), bytes.NewReader(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return client.Do(req)
}

// WaitForNotifications waits for at least the specified number of notifications
func (h *TestHarness) WaitForNotifications(count int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		total := h.SlackNotifier.GetNotificationCount() + h.PagerDutyNotifier.GetNotificationCount()
		if total >= count {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// WaitForSlackNotification waits for a Slack notification with the given fingerprint
func (h *TestHarness) WaitForSlackNotification(fingerprint string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if h.SlackNotifier.HasNotificationWithFingerprint(fingerprint) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// WaitForPagerDutyNotification waits for a PagerDuty notification with the given fingerprint
func (h *TestHarness) WaitForPagerDutyNotification(fingerprint string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if h.PagerDutyNotifier.HasNotificationWithFingerprint(fingerprint) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// GetFreePort returns an available port for testing
func GetFreePort() (int, error) {
	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
