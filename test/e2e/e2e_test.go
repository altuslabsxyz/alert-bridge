// Package e2e provides in-process end-to-end tests using mock notifiers.
// These tests run without Docker and are much faster than Docker-based e2e tests.
package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/entity"
	"github.com/qj0r9j0vc2/alert-bridge/test/e2e/harness"
)

// TestAlertCreationSlack tests that alerts are delivered to Slack notifier
func TestAlertCreationSlack(t *testing.T) {
	h := harness.NewTestHarness(t)

	// Create and send a test alert
	alert := harness.CreateTestAlert("high_cpu_critical", nil)
	resp, err := h.SendAlert([]harness.AlertmanagerAlert{alert})
	if err != nil {
		t.Fatalf("Failed to send alert: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// Wait for notification to be processed
	if !h.WaitForSlackNotification(alert.Fingerprint, 5*time.Second) {
		t.Fatal("Slack notification was not received within timeout")
	}

	// Verify the notification was recorded
	notifications := h.SlackNotifier.GetNotificationsByFingerprint(alert.Fingerprint)
	if len(notifications) == 0 {
		t.Fatal("Expected at least one Slack notification")
	}

	notification := notifications[0]

	// Verify notification properties
	if notification.Name != "HighCPU" {
		t.Errorf("Expected alert name 'HighCPU', got '%s'", notification.Name)
	}

	if notification.Severity != entity.SeverityCritical {
		t.Errorf("Expected severity critical, got %v", notification.Severity)
	}

	if notification.State != entity.StateActive {
		t.Errorf("Expected state firing, got %v", notification.State)
	}

	t.Logf("Alert successfully delivered to Slack mock: messageID=%s", notification.MessageID)
}

// TestAlertCreationPagerDuty tests that alerts are delivered to PagerDuty notifier
func TestAlertCreationPagerDuty(t *testing.T) {
	h := harness.NewTestHarness(t)

	// Create and send a test alert
	alert := harness.CreateTestAlert("service_down_critical", nil)
	resp, err := h.SendAlert([]harness.AlertmanagerAlert{alert})
	if err != nil {
		t.Fatalf("Failed to send alert: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// Wait for notification
	if !h.WaitForPagerDutyNotification(alert.Fingerprint, 5*time.Second) {
		t.Fatal("PagerDuty notification was not received within timeout")
	}

	// Verify the notification
	notifications := h.PagerDutyNotifier.GetNotificationsByFingerprint(alert.Fingerprint)
	if len(notifications) == 0 {
		t.Fatal("Expected at least one PagerDuty notification")
	}

	notification := notifications[0]

	if notification.Severity != entity.SeverityCritical {
		t.Errorf("Expected severity critical, got %v", notification.Severity)
	}

	t.Logf("Alert successfully delivered to PagerDuty mock: messageID=%s", notification.MessageID)
}

// TestAlertDeduplication tests that duplicate alerts are not sent multiple times
func TestAlertDeduplication(t *testing.T) {
	h := harness.NewTestHarness(t)

	// Create and send first alert
	alert := harness.CreateTestAlert("duplicate_test_alert", nil)
	resp, err := h.SendAlert([]harness.AlertmanagerAlert{alert})
	if err != nil {
		t.Fatalf("Failed to send first alert: %v", err)
	}
	resp.Body.Close()

	// Wait for first notification
	if !h.WaitForSlackNotification(alert.Fingerprint, 5*time.Second) {
		t.Fatal("First notification not received")
	}

	initialSlackCount := h.SlackNotifier.GetNotificationCount()
	initialPDCount := h.PagerDutyNotifier.GetNotificationCount()

	t.Logf("After first alert: Slack=%d, PagerDuty=%d", initialSlackCount, initialPDCount)

	// Send duplicate alert (same fingerprint)
	resp, err = h.SendAlert([]harness.AlertmanagerAlert{alert})
	if err != nil {
		t.Fatalf("Failed to send duplicate alert: %v", err)
	}
	resp.Body.Close()

	// Wait a bit to ensure processing
	time.Sleep(100 * time.Millisecond)

	// Verify no new notifications were sent
	finalSlackCount := h.SlackNotifier.GetNotificationCount()
	finalPDCount := h.PagerDutyNotifier.GetNotificationCount()

	t.Logf("After duplicate: Slack=%d, PagerDuty=%d", finalSlackCount, finalPDCount)

	if finalSlackCount != initialSlackCount {
		t.Errorf("Expected %d Slack notifications after dedup, got %d", initialSlackCount, finalSlackCount)
	}

	if finalPDCount != initialPDCount {
		t.Errorf("Expected %d PagerDuty notifications after dedup, got %d", initialPDCount, finalPDCount)
	}

	t.Log("Alert deduplication working correctly")
}

// TestAlertResolution tests alert resolution notifications
func TestAlertResolution(t *testing.T) {
	h := harness.NewTestHarness(t)

	// Send firing alert
	alert := harness.CreateTestAlert("memory_pressure_warning", map[string]string{
		"test": "resolution",
	})
	resp, err := h.SendAlert([]harness.AlertmanagerAlert{alert})
	if err != nil {
		t.Fatalf("Failed to send firing alert: %v", err)
	}
	resp.Body.Close()

	// Wait for initial notification
	if !h.WaitForSlackNotification(alert.Fingerprint, 5*time.Second) {
		t.Fatal("Initial notification not received")
	}

	// Verify initial state is firing
	notifications := h.SlackNotifier.GetNotificationsByFingerprint(alert.Fingerprint)
	if len(notifications) == 0 {
		t.Fatal("Expected initial notification")
	}

	if notifications[0].State != entity.StateActive {
		t.Errorf("Expected initial state firing, got %v", notifications[0].State)
	}

	// Send resolved alert
	resolvedAlert := harness.CreateResolvedAlert(alert)
	resp, err = h.SendAlert([]harness.AlertmanagerAlert{resolvedAlert})
	if err != nil {
		t.Fatalf("Failed to send resolved alert: %v", err)
	}
	resp.Body.Close()

	// Wait for update notification
	time.Sleep(200 * time.Millisecond)

	// Verify we got an update notification (not a new one)
	notifications = h.SlackNotifier.GetNotificationsByFingerprint(alert.Fingerprint)
	if len(notifications) < 2 {
		t.Fatalf("Expected at least 2 notifications (initial + update), got %d", len(notifications))
	}

	// The last notification should be an update with resolved state
	lastNotification := notifications[len(notifications)-1]
	if !lastNotification.IsUpdate {
		t.Error("Expected last notification to be an update")
	}

	if lastNotification.State != entity.StateResolved {
		t.Errorf("Expected resolved state, got %v", lastNotification.State)
	}

	t.Log("Alert resolution notifications working correctly")
}

// TestMultipleAlertsInBatch tests processing multiple alerts in a single webhook
func TestMultipleAlertsInBatch(t *testing.T) {
	h := harness.NewTestHarness(t)

	// Create multiple alerts
	alert1 := harness.CreateTestAlert("high_cpu_critical", map[string]string{"test": "batch_1"})
	alert2 := harness.CreateTestAlert("disk_space_critical", map[string]string{"test": "batch_2"})
	alert3 := harness.CreateTestAlert("backup_failed_critical", map[string]string{"test": "batch_3"})

	// Send all alerts in one batch
	resp, err := h.SendAlert([]harness.AlertmanagerAlert{alert1, alert2, alert3})
	if err != nil {
		t.Fatalf("Failed to send alerts: %v", err)
	}
	resp.Body.Close()

	// Wait for all notifications (6 total: 3 Slack + 3 PagerDuty)
	if !h.WaitForNotifications(6, 10*time.Second) {
		t.Fatalf("Not all notifications received. Slack=%d, PagerDuty=%d",
			h.SlackNotifier.GetNotificationCount(),
			h.PagerDutyNotifier.GetNotificationCount())
	}

	// Verify each alert was delivered to both notifiers
	for _, fingerprint := range []string{alert1.Fingerprint, alert2.Fingerprint, alert3.Fingerprint} {
		if !h.SlackNotifier.HasNotificationWithFingerprint(fingerprint) {
			t.Errorf("Slack notification missing for fingerprint %s", fingerprint)
		}
		if !h.PagerDutyNotifier.HasNotificationWithFingerprint(fingerprint) {
			t.Errorf("PagerDuty notification missing for fingerprint %s", fingerprint)
		}
	}

	t.Logf("Multiple alerts handled correctly: Slack=%d, PagerDuty=%d",
		h.SlackNotifier.GetNotificationCount(),
		h.PagerDutyNotifier.GetNotificationCount())
}

// TestDifferentSeverityLevels tests alerts with different severity levels
func TestDifferentSeverityLevels(t *testing.T) {
	h := harness.NewTestHarness(t)

	// Send critical alert
	criticalAlert := harness.CreateTestAlert("high_cpu_critical", map[string]string{"test": "severity_critical"})
	resp, err := h.SendAlert([]harness.AlertmanagerAlert{criticalAlert})
	if err != nil {
		t.Fatalf("Failed to send critical alert: %v", err)
	}
	resp.Body.Close()

	// Wait for critical notification
	if !h.WaitForPagerDutyNotification(criticalAlert.Fingerprint, 5*time.Second) {
		t.Fatal("Critical alert notification not received")
	}

	// Send warning alert
	warningAlert := harness.CreateTestAlert("memory_pressure_warning", map[string]string{"test": "severity_warning"})
	resp, err = h.SendAlert([]harness.AlertmanagerAlert{warningAlert})
	if err != nil {
		t.Fatalf("Failed to send warning alert: %v", err)
	}
	resp.Body.Close()

	// Wait for warning notification
	if !h.WaitForPagerDutyNotification(warningAlert.Fingerprint, 5*time.Second) {
		t.Fatal("Warning alert notification not received")
	}

	// Verify critical severity
	criticalNotifications := h.PagerDutyNotifier.GetNotificationsByFingerprint(criticalAlert.Fingerprint)
	if len(criticalNotifications) == 0 {
		t.Fatal("No critical notifications found")
	}
	if criticalNotifications[0].Severity != entity.SeverityCritical {
		t.Errorf("Expected critical severity, got %v", criticalNotifications[0].Severity)
	}

	// Verify warning severity
	warningNotifications := h.PagerDutyNotifier.GetNotificationsByFingerprint(warningAlert.Fingerprint)
	if len(warningNotifications) == 0 {
		t.Fatal("No warning notifications found")
	}
	if warningNotifications[0].Severity != entity.SeverityWarning {
		t.Errorf("Expected warning severity, got %v", warningNotifications[0].Severity)
	}

	t.Log("Different severity levels handled correctly")
}

// TestNotifierFailureHandling tests behavior when a notifier fails
func TestNotifierFailureHandling(t *testing.T) {
	h := harness.NewTestHarness(t)

	// Configure Slack to fail on next call
	h.SlackNotifier.SetFailNext(errMockNotifierFailure)

	// Send alert
	alert := harness.CreateTestAlert("high_cpu_critical", map[string]string{"test": "failure"})
	resp, err := h.SendAlert([]harness.AlertmanagerAlert{alert})
	if err != nil {
		t.Fatalf("Failed to send alert: %v", err)
	}
	resp.Body.Close()

	// Wait for PagerDuty notification (Slack should have failed)
	if !h.WaitForPagerDutyNotification(alert.Fingerprint, 5*time.Second) {
		t.Fatal("PagerDuty notification not received despite Slack failure")
	}

	// Verify Slack didn't receive notification (it failed)
	slackNotifications := h.SlackNotifier.GetNotificationsByFingerprint(alert.Fingerprint)
	if len(slackNotifications) != 0 {
		t.Errorf("Expected no Slack notifications due to failure, got %d", len(slackNotifications))
	}

	// Verify PagerDuty did receive notification
	pdNotifications := h.PagerDutyNotifier.GetNotificationsByFingerprint(alert.Fingerprint)
	if len(pdNotifications) == 0 {
		t.Error("Expected PagerDuty notification despite Slack failure")
	}

	t.Log("Notifier failure handled correctly - other notifiers still received alerts")
}

// TestHealthEndpoint tests the health endpoint
func TestHealthEndpoint(t *testing.T) {
	h := harness.NewTestHarness(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.ServerURL()+"/health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	t.Log("Health endpoint working correctly")
}

// errMockNotifierFailure is a test error for simulating notifier failures
var errMockNotifierFailure = &mockNotifierError{message: "simulated notifier failure"}

type mockNotifierError struct {
	message string
}

func (e *mockNotifierError) Error() string {
	return e.message
}
