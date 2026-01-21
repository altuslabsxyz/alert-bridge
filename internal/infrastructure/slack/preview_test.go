package slack

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/entity"
)

func TestPreviewBlocks(t *testing.T) {
	builder := NewMessageBuilder(nil)

	alert := &entity.Alert{
		ID:          "test-123",
		Name:        "HighMemoryUsage",
		Summary:     "Memory usage exceeded 90% threshold",
		Severity:    entity.SeverityCritical,
		State:       entity.StateActive,
		Instance:    "server-prod-01",
		Target:      "memory",
		Fingerprint: "abc123def456",
		FiredAt:     time.Now().Add(-30 * time.Minute),
	}

	blocks := builder.BuildAlertMessage(alert)

	jsonBytes, _ := json.MarshalIndent(map[string]interface{}{
		"blocks": blocks,
	}, "", "  ")

	fmt.Println("\n========== SLACK BLOCK KIT JSON ==========")
	fmt.Println(string(jsonBytes))
	fmt.Println("===========================================")
	fmt.Println("\nCopy this JSON to https://app.slack.com/block-kit-builder")
}
