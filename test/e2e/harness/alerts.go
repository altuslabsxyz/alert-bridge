package harness

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"time"

	"github.com/qj0r9j0vc2/alert-bridge/internal/adapter/dto"
)

// AlertmanagerAlert is an alias for dto.AlertmanagerAlert for external use
type AlertmanagerAlert = dto.AlertmanagerAlert

// AlertFixtures contains pre-defined alert configurations for testing
var AlertFixtures = map[string]AlertFixture{
	"high_cpu_critical": {
		Labels: map[string]string{
			"alertname": "HighCPU",
			"severity":  "critical",
			"instance":  "server-01",
			"job":       "test-service",
			"component": "cpu",
			"service":   "test-service",
		},
		Annotations: map[string]string{
			"summary":     "High CPU usage detected on server-01",
			"description": "CPU usage is above 90% for more than 5 minutes on server-01",
			"runbook_url": "https://wiki.example.com/runbooks/high-cpu",
		},
	},
	"memory_pressure_warning": {
		Labels: map[string]string{
			"alertname": "MemoryPressure",
			"severity":  "warning",
			"instance":  "server-02",
			"job":       "test-service",
			"component": "memory",
			"service":   "test-service",
		},
		Annotations: map[string]string{
			"summary":     "Memory pressure detected on server-02",
			"description": "Available memory is below 10% on server-02",
		},
	},
	"service_down_critical": {
		Labels: map[string]string{
			"alertname": "ServiceDown",
			"severity":  "critical",
			"instance":  "server-03",
			"job":       "alert-bridge",
			"component": "availability",
		},
		Annotations: map[string]string{
			"summary":     "Service alert-bridge is down",
			"description": "The service alert-bridge on server-03 has been down for more than 30 seconds",
		},
	},
	"duplicate_test_alert": {
		Labels: map[string]string{
			"alertname":     "DuplicateAlert",
			"severity":      "warning",
			"instance":      "server-01",
			"job":           "test-target",
			"test_scenario": "deduplication",
		},
		Annotations: map[string]string{
			"summary":     "Test alert for deduplication scenario",
			"description": "This alert is used to test deduplication logic",
		},
	},
	"disk_space_critical": {
		Labels: map[string]string{
			"alertname": "DiskSpaceLow",
			"severity":  "critical",
			"instance":  "server-04",
			"job":       "infrastructure",
			"component": "storage",
			"service":   "database",
		},
		Annotations: map[string]string{
			"summary":     "Disk space critically low on server-04",
			"description": "Less than 5% disk space remaining on /data partition",
			"runbook_url": "https://wiki.example.com/runbooks/disk-space",
		},
	},
	"backup_failed_critical": {
		Labels: map[string]string{
			"alertname": "BackupFailed",
			"severity":  "critical",
			"instance":  "backup-server",
			"job":       "backup",
			"component": "data-protection",
		},
		Annotations: map[string]string{
			"summary":     "Backup operation failed",
			"description": "Database backup failed for the second consecutive day",
			"runbook_url": "https://wiki.example.com/runbooks/backup-failure",
		},
	},
}

// AlertFixture defines the template for a test alert
type AlertFixture struct {
	Labels      map[string]string
	Annotations map[string]string
}

// CreateTestAlert creates a test alert from a fixture
func CreateTestAlert(fixtureName string, overrideLabels map[string]string) dto.AlertmanagerAlert {
	fixture, ok := AlertFixtures[fixtureName]
	if !ok {
		panic(fmt.Sprintf("unknown fixture: %s", fixtureName))
	}

	// Copy labels
	labels := make(map[string]string)
	for k, v := range fixture.Labels {
		labels[k] = v
	}

	// Apply overrides
	for k, v := range overrideLabels {
		labels[k] = v
	}

	// Copy annotations
	annotations := make(map[string]string)
	for k, v := range fixture.Annotations {
		annotations[k] = v
	}

	now := time.Now()

	return dto.AlertmanagerAlert{
		Status:      "firing",
		Labels:      labels,
		Annotations: annotations,
		StartsAt:    now,
		Fingerprint: GenerateFingerprint(labels),
	}
}

// CreateResolvedAlert creates a resolved version of an alert
func CreateResolvedAlert(alert dto.AlertmanagerAlert) dto.AlertmanagerAlert {
	resolved := alert
	resolved.Status = "resolved"
	resolved.EndsAt = time.Now()
	return resolved
}

// GenerateFingerprint generates a fingerprint for an alert based on labels
func GenerateFingerprint(labels map[string]string) string {
	// Sort labels and create deterministic string
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	hash := sha256.New()
	for _, k := range keys {
		hash.Write([]byte(k))
		hash.Write([]byte(labels[k]))
	}

	return fmt.Sprintf("%x", hash.Sum(nil))[:16]
}
