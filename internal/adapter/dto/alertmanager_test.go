package dto

import (
	"testing"
	"time"

	"github.com/altuslabsxyz/alert-bridge/internal/domain/entity"
)

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected entity.AlertSeverity
	}{
		// Critical mappings
		{"critical maps to SeverityCritical", "critical", entity.SeverityCritical},
		{"page maps to SeverityCritical", "page", entity.SeverityCritical},
		// Warning mappings
		{"warning maps to SeverityWarning", "warning", entity.SeverityWarning},
		{"warn maps to SeverityWarning", "warn", entity.SeverityWarning},
		// Default/Info mappings
		{"info maps to SeverityInfo", "info", entity.SeverityInfo},
		{"empty string maps to SeverityInfo", "", entity.SeverityInfo},
		{"unknown value maps to SeverityInfo", "unknown", entity.SeverityInfo},
		{"error maps to SeverityInfo", "error", entity.SeverityInfo},
		// Case sensitivity (should not match)
		{"Critical (uppercase) maps to SeverityInfo", "Critical", entity.SeverityInfo},
		{"WARNING (uppercase) maps to SeverityInfo", "WARNING", entity.SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("mapSeverity(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToProcessAlertInput(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name     string
		alert    AlertmanagerAlert
		validate func(t *testing.T, result ProcessAlertInput)
	}{
		{
			name: "converts all fields correctly",
			alert: AlertmanagerAlert{
				Fingerprint: "abc123",
				Status:      "firing",
				Labels: map[string]string{
					"alertname": "HighCPU",
					"instance":  "server-1",
					"job":       "node-exporter",
					"severity":  "critical",
				},
				Annotations: map[string]string{
					"summary":     "CPU usage is high",
					"description": "CPU usage exceeded 90%",
				},
				StartsAt: now,
			},
			validate: func(t *testing.T, result ProcessAlertInput) {
				if result.Fingerprint != "abc123" {
					t.Errorf("Fingerprint = %q, want %q", result.Fingerprint, "abc123")
				}
				if result.Name != "HighCPU" {
					t.Errorf("Name = %q, want %q", result.Name, "HighCPU")
				}
				if result.Instance != "server-1" {
					t.Errorf("Instance = %q, want %q", result.Instance, "server-1")
				}
				if result.Target != "node-exporter" {
					t.Errorf("Target = %q, want %q", result.Target, "node-exporter")
				}
				if result.Summary != "CPU usage is high" {
					t.Errorf("Summary = %q, want %q", result.Summary, "CPU usage is high")
				}
				if result.Description != "CPU usage exceeded 90%" {
					t.Errorf("Description = %q, want %q", result.Description, "CPU usage exceeded 90%")
				}
				if result.Severity != entity.SeverityCritical {
					t.Errorf("Severity = %v, want %v", result.Severity, entity.SeverityCritical)
				}
				if result.Status != "firing" {
					t.Errorf("Status = %q, want %q", result.Status, "firing")
				}
				if !result.FiredAt.Equal(now) {
					t.Errorf("FiredAt = %v, want %v", result.FiredAt, now)
				}
			},
		},
		{
			name: "handles missing labels gracefully",
			alert: AlertmanagerAlert{
				Fingerprint: "def456",
				Status:      "resolved",
				Labels:      map[string]string{},
				Annotations: map[string]string{},
				StartsAt:    now,
			},
			validate: func(t *testing.T, result ProcessAlertInput) {
				if result.Name != "" {
					t.Errorf("Name = %q, want empty string", result.Name)
				}
				if result.Instance != "" {
					t.Errorf("Instance = %q, want empty string", result.Instance)
				}
				if result.Target != "" {
					t.Errorf("Target = %q, want empty string", result.Target)
				}
				if result.Severity != entity.SeverityInfo {
					t.Errorf("Severity = %v, want %v", result.Severity, entity.SeverityInfo)
				}
			},
		},
		{
			name: "preserves original labels and annotations maps",
			alert: AlertmanagerAlert{
				Labels: map[string]string{
					"alertname": "Test",
					"custom":    "value",
				},
				Annotations: map[string]string{
					"summary": "Test summary",
					"runbook": "http://example.com/runbook",
				},
			},
			validate: func(t *testing.T, result ProcessAlertInput) {
				if result.Labels["custom"] != "value" {
					t.Errorf("Labels[custom] = %q, want %q", result.Labels["custom"], "value")
				}
				if result.Annotations["runbook"] != "http://example.com/runbook" {
					t.Errorf("Annotations[runbook] = %q, want %q", result.Annotations["runbook"], "http://example.com/runbook")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToProcessAlertInput(tt.alert)
			tt.validate(t, result)
		})
	}
}

func TestAlertmanagerAlert_IsFiring(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"firing", true},
		{"resolved", false},
		{"", false},
		{"unknown", false},
		{"Firing", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			alert := &AlertmanagerAlert{Status: tt.status}
			if got := alert.IsFiring(); got != tt.expected {
				t.Errorf("IsFiring() = %v, want %v for status %q", got, tt.expected, tt.status)
			}
		})
	}
}

func TestAlertmanagerAlert_IsResolved(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"resolved", true},
		{"firing", false},
		{"", false},
		{"unknown", false},
		{"Resolved", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			alert := &AlertmanagerAlert{Status: tt.status}
			if got := alert.IsResolved(); got != tt.expected {
				t.Errorf("IsResolved() = %v, want %v for status %q", got, tt.expected, tt.status)
			}
		})
	}
}
