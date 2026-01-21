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
		{"critical", "critical", entity.SeverityCritical},
		{"page", "page", entity.SeverityCritical},
		{"warning", "warning", entity.SeverityWarning},
		{"warn", "warn", entity.SeverityWarning},
		{"info", "info", entity.SeverityInfo},
		{"empty", "", entity.SeverityInfo},
		{"unknown", "unknown", entity.SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapSeverity(tt.input)
			if got != tt.expected {
				t.Errorf("mapSeverity(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestToProcessAlertInput(t *testing.T) {
	now := time.Now()

	alert := AlertmanagerAlert{
		Status:      "firing",
		Fingerprint: "abc123",
		Labels: map[string]string{
			"alertname": "HighCPU",
			"instance":  "server1:9090",
			"job":       "prometheus",
			"severity":  "critical",
		},
		Annotations: map[string]string{
			"summary":     "CPU usage is high",
			"description": "CPU usage exceeded 90%",
		},
		StartsAt: now,
	}

	input := ToProcessAlertInput(alert)

	if input.Fingerprint != "abc123" {
		t.Errorf("Fingerprint = %v, want %v", input.Fingerprint, "abc123")
	}
	if input.Name != "HighCPU" {
		t.Errorf("Name = %v, want %v", input.Name, "HighCPU")
	}
	if input.Instance != "server1:9090" {
		t.Errorf("Instance = %v, want %v", input.Instance, "server1:9090")
	}
	if input.Target != "prometheus" {
		t.Errorf("Target = %v, want %v", input.Target, "prometheus")
	}
	if input.Summary != "CPU usage is high" {
		t.Errorf("Summary = %v, want %v", input.Summary, "CPU usage is high")
	}
	if input.Severity != entity.SeverityCritical {
		t.Errorf("Severity = %v, want %v", input.Severity, entity.SeverityCritical)
	}
	if input.Status != "firing" {
		t.Errorf("Status = %v, want %v", input.Status, "firing")
	}
}

func TestAlertmanagerAlert_IsFiring(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"firing", "firing", true},
		{"resolved", "resolved", false},
		{"other", "pending", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := AlertmanagerAlert{Status: tt.status}
			if got := a.IsFiring(); got != tt.want {
				t.Errorf("IsFiring() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlertmanagerAlert_IsResolved(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"resolved", "resolved", true},
		{"firing", "firing", false},
		{"other", "pending", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := AlertmanagerAlert{Status: tt.status}
			if got := a.IsResolved(); got != tt.want {
				t.Errorf("IsResolved() = %v, want %v", got, tt.want)
			}
		})
	}
}
