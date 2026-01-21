# Alert-Bridge Mock-Based E2E Tests

Fast, in-process end-to-end tests using mock notifiers. These tests run without Docker and complete in under 1 second.

## Quick Start

```bash
# Run mock-based e2e tests
make test-e2e

# Or directly with go test
go test -v ./test/e2e/... -timeout 60s
```

## Architecture

Unlike the Docker-based E2E tests (see `test/e2e-docker/`), these tests run entirely in-process:

```
┌─────────────────────────────────────────────────────────────┐
│                    Go Test Process                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐      ┌──────────────────────────────────┐│
│  │ Test Code    │─────▶│ TestHarness                      ││
│  │ e2e_test.go  │      │ ├── In-memory AlertRepository    ││
│  └──────────────┘      │ ├── ProcessAlertUseCase          ││
│                        │ ├── AlertmanagerHandler          ││
│                        │ └── httptest.Server              ││
│                        └─────────────┬────────────────────┘│
│                                      │                      │
│                         ┌────────────┴────────────┐        │
│                         ▼                         ▼        │
│                  ┌──────────────┐         ┌──────────────┐ │
│                  │ MockNotifier │         │ MockNotifier │ │
│                  │ (Slack)      │         │ (PagerDuty)  │ │
│                  └──────────────┘         └──────────────┘ │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Test Scenarios

| Test | Description |
|------|-------------|
| `TestAlertCreationSlack` | Verifies alerts are delivered to Slack notifier |
| `TestAlertCreationPagerDuty` | Verifies alerts are delivered to PagerDuty notifier |
| `TestAlertDeduplication` | Tests that duplicate alerts are suppressed |
| `TestAlertResolution` | Tests alert resolution notifications |
| `TestMultipleAlertsInBatch` | Tests processing multiple alerts in one webhook |
| `TestDifferentSeverityLevels` | Tests critical vs warning severity handling |
| `TestNotifierFailureHandling` | Tests resilience when one notifier fails |
| `TestHealthEndpoint` | Verifies health endpoint responds correctly |

## Components

### Test Harness (`harness/harness.go`)

The test harness sets up a complete Alert-Bridge environment in-process:

```go
h := harness.NewTestHarness(t)

// Access mock notifiers directly
h.SlackNotifier.GetNotifications()
h.PagerDutyNotifier.GetNotificationsByFingerprint(fp)

// Send alerts via HTTP
resp, err := h.SendAlert([]harness.AlertmanagerAlert{alert})

// Wait for async notifications
h.WaitForSlackNotification(fingerprint, 5*time.Second)
```

### Mock Notifier (`mocks/notifier.go`)

Thread-safe mock implementation of `alert.Notifier` interface:

```go
// Records all notifications for assertions
notifications := mock.GetNotificationsByFingerprint(fingerprint)

// Simulate failures
mock.SetFailNext(errors.New("connection refused"))

// Reset state between tests
mock.Reset()
```

### Alert Fixtures (`harness/alerts.go`)

Pre-defined alert templates for testing:

```go
alert := harness.CreateTestAlert("high_cpu_critical", nil)

// Override labels
alert := harness.CreateTestAlert("high_cpu_critical", map[string]string{
    "severity": "warning",
})

// Create resolved version
resolved := harness.CreateResolvedAlert(alert)
```

## Comparison: Mock vs Docker E2E Tests

| Aspect | Mock-Based (`test/e2e/`) | Docker-Based (`test/e2e-docker/`) |
|--------|--------------------------|-----------------------------------|
| **Speed** | ~1 second | ~5 minutes |
| **Dependencies** | None | Docker, Docker Compose |
| **Network** | In-process | Real HTTP |
| **Isolation** | Per-test harness | Shared containers |
| **CI Integration** | Easy | Requires Docker |
| **Use Case** | Fast feedback loop | Integration validation |

## Writing New Tests

```go
func TestMyScenario(t *testing.T) {
    // Create test harness (auto-cleanup on test end)
    h := harness.NewTestHarness(t)

    // Create alert from fixture
    alert := harness.CreateTestAlert("high_cpu_critical", map[string]string{
        "custom_label": "value",
    })

    // Send to webhook endpoint
    resp, err := h.SendAlert([]harness.AlertmanagerAlert{alert})
    if err != nil {
        t.Fatalf("Failed to send alert: %v", err)
    }
    defer resp.Body.Close()

    // Wait for async notification
    if !h.WaitForSlackNotification(alert.Fingerprint, 5*time.Second) {
        t.Fatal("Notification not received")
    }

    // Assert on recorded notifications
    notifications := h.SlackNotifier.GetNotificationsByFingerprint(alert.Fingerprint)
    if len(notifications) != 1 {
        t.Errorf("Expected 1 notification, got %d", len(notifications))
    }
}
```

## Available Fixtures

| Fixture Name | Severity | Description |
|--------------|----------|-------------|
| `high_cpu_critical` | critical | High CPU usage on server-01 |
| `memory_pressure_warning` | warning | Memory pressure on server-02 |
| `service_down_critical` | critical | Service unavailability |
| `duplicate_test_alert` | warning | Used for deduplication tests |
| `disk_space_critical` | critical | Low disk space |
| `backup_failed_critical` | critical | Backup operation failure |

## Troubleshooting

### Tests timeout waiting for notifications

Notifications are processed synchronously. If tests timeout, check:
1. The harness was created correctly
2. The alert payload is valid
3. No panics in the test output

### Deduplication tests fail

Each test should use unique label combinations to avoid fingerprint collisions:

```go
// Good: unique labels per test
alert := harness.CreateTestAlert("high_cpu_critical", map[string]string{
    "test": "my_unique_test_name",
})
```

## Directory Structure

```
test/e2e/
├── README.md              ← This file
├── e2e_test.go            ← Test scenarios
├── harness/
│   ├── harness.go         ← Test harness setup
│   └── alerts.go          ← Alert fixtures and helpers
└── mocks/
    └── notifier.go        ← Mock Notifier implementation
```

## For Docker-Based E2E Tests

For comprehensive integration testing with real Docker containers, see `test/e2e-docker/README.md`.
