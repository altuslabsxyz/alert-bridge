package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/qj0r9j0vc2/alert-bridge/internal/adapter/dto"
	"github.com/qj0r9j0vc2/alert-bridge/internal/usecase/alert"
	pdUseCase "github.com/qj0r9j0vc2/alert-bridge/internal/usecase/pagerduty"
)

// PagerDutyWebhookHandler handles PagerDuty V3 webhook events.
type PagerDutyWebhookHandler struct {
	handleWebhook *pdUseCase.HandleWebhookUseCase
	webhookSecret string
	logger        alert.Logger
}

// NewPagerDutyWebhookHandler creates a new PagerDuty webhook handler.
func NewPagerDutyWebhookHandler(
	handleWebhook *pdUseCase.HandleWebhookUseCase,
	webhookSecret string,
	logger alert.Logger,
) *PagerDutyWebhookHandler {
	return &PagerDutyWebhookHandler{
		handleWebhook: handleWebhook,
		webhookSecret: webhookSecret,
		logger:        logger,
	}
}

// ServeHTTP handles POST /webhook/pagerduty
func (h *PagerDutyWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", "error", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Verify webhook signature if secret is configured
	if h.webhookSecret != "" {
		signatures := r.Header.Values("X-PagerDuty-Signature")
		if !h.verifySignature(body, signatures) {
			h.logger.Warn("invalid PagerDuty webhook signature")
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Parse webhook payload
	var payload dto.PagerDutyWebhookV3
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("failed to parse PagerDuty webhook payload", "error", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	var processed, skipped int

	// Process each message
	for _, msg := range payload.Messages {
		event := msg.Event

		// Skip unsupported event types
		if !dto.IsSupportedEventType(event.EventType) {
			h.logger.Debug("skipping unsupported event type",
				"eventType", event.EventType,
			)
			skipped++
			continue
		}

		// Build input
		input := dto.HandlePagerDutyWebhookInput{
			EventType:     event.EventType,
			IncidentID:    event.Data.ID,
			IncidentKey:   event.Data.IncidentKey,
			Status:        event.Data.Status,
			ResolveReason: event.Data.ResolveReason,
		}

		// Extract user info from agent or last status change
		if event.Agent != nil {
			input.UserID = event.Agent.ID
			input.UserEmail = event.Agent.Email
			input.UserName = event.Agent.Name
		} else if event.Data.LastStatusChangeBy != nil {
			input.UserID = event.Data.LastStatusChangeBy.ID
			input.UserEmail = event.Data.LastStatusChangeBy.Email
			input.UserName = event.Data.LastStatusChangeBy.Summary
		}

		// For acknowledged events, try to get acknowledger info
		if event.EventType == "incident.acknowledged" && len(event.Data.Acknowledgers) > 0 {
			acker := event.Data.Acknowledgers[len(event.Data.Acknowledgers)-1].Acknowledger
			input.UserID = acker.ID
			input.UserEmail = acker.Email
			input.UserName = acker.Summary
		}

		// Execute use case
		output, err := h.handleWebhook.Execute(ctx, input)
		if err != nil {
			h.logger.Error("failed to handle PagerDuty webhook",
				"eventType", event.EventType,
				"incidentID", event.Data.ID,
				"error", err,
			)
			continue
		}

		if output.Processed {
			processed++
			h.logger.Info("PagerDuty webhook processed",
				"eventType", event.EventType,
				"incidentID", event.Data.ID,
				"alertID", output.AlertID,
				"message", output.Message,
			)
		} else {
			skipped++
		}
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "ok",
		"processed": processed,
		"skipped":   skipped,
	})
}

// verifySignature verifies the PagerDuty webhook signature.
// PagerDuty sends multiple signatures with different versions.
func (h *PagerDutyWebhookHandler) verifySignature(body []byte, signatures []string) bool {
	if len(signatures) == 0 {
		return false
	}

	for _, sig := range signatures {
		// Parse version and signature
		// Format: "v1=<signature>"
		parts := strings.SplitN(sig, "=", 2)
		if len(parts) != 2 {
			continue
		}

		version := parts[0]
		signature := parts[1]

		// Only support v1 signatures
		if version != "v1" {
			continue
		}

		// Compute expected signature
		mac := hmac.New(sha256.New, []byte(h.webhookSecret))
		mac.Write(body)
		expectedSig := hex.EncodeToString(mac.Sum(nil))

		// Compare signatures
		if hmac.Equal([]byte(signature), []byte(expectedSig)) {
			return true
		}
	}

	return false
}

// extractUserFromAcknowledgers extracts the user from the acknowledgers list.
func extractUserFromAcknowledgers(acknowledgers []dto.PagerDutyAcknowledgerRef) (userID, userEmail, userName string) {
	if len(acknowledgers) == 0 {
		return "", "", ""
	}

	// Get the most recent acknowledger
	acker := acknowledgers[len(acknowledgers)-1].Acknowledger
	return acker.ID, acker.Email, acker.Summary
}
