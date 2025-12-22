package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"

	"github.com/qj0r9j0vc2/alert-bridge/internal/adapter/dto"
	"github.com/qj0r9j0vc2/alert-bridge/internal/usecase/alert"
	slackUseCase "github.com/qj0r9j0vc2/alert-bridge/internal/usecase/slack"
)

// SlackInteractionHandler handles Slack interactive component callbacks.
type SlackInteractionHandler struct {
	handleInteraction *slackUseCase.HandleInteractionUseCase
	signingSecret     string
	logger            alert.Logger
}

// NewSlackInteractionHandler creates a new Slack interaction handler.
func NewSlackInteractionHandler(
	handleInteraction *slackUseCase.HandleInteractionUseCase,
	signingSecret string,
	logger alert.Logger,
) *SlackInteractionHandler {
	return &SlackInteractionHandler{
		handleInteraction: handleInteraction,
		signingSecret:     signingSecret,
		logger:            logger,
	}
}

// ServeHTTP handles POST /webhook/slack/interaction
func (h *SlackInteractionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body for signature verification
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", "error", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	// Verify Slack signature
	if err := h.verifySlackSignature(r.Header, body); err != nil {
		h.logger.Warn("invalid slack signature", "error", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse the payload
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", "error", err)
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	payloadStr := r.FormValue("payload")
	if payloadStr == "" {
		http.Error(w, "missing payload", http.StatusBadRequest)
		return
	}

	var payload slack.InteractionCallback
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		h.logger.Error("failed to parse interaction payload", "error", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Handle block actions
	for _, action := range payload.ActionCallback.BlockActions {
		input := dto.SlackInteractionInput{
			ActionID:    action.ActionID,
			AlertID:     action.Value,
			UserID:      payload.User.ID,
			UserName:    payload.User.Name,
			ResponseURL: payload.ResponseURL,
			ChannelID:   payload.Channel.ID,
			MessageTS:   payload.Message.Timestamp,
			TriggerID:   payload.TriggerID,
		}

		// Get value from static select if present
		if action.SelectedOption.Value != "" {
			input.Value = action.SelectedOption.Value
		}

		output, err := h.handleInteraction.Execute(ctx, input)
		if err != nil {
			h.logger.Error("failed to handle interaction",
				"actionID", action.ActionID,
				"userID", payload.User.ID,
				"error", err,
			)
			// Continue processing other actions
			continue
		}

		h.logger.Info("interaction handled",
			"actionID", action.ActionID,
			"userID", payload.User.ID,
			"success", output.Success,
			"message", output.Message,
		)
	}

	// Acknowledge the interaction immediately
	w.WriteHeader(http.StatusOK)
}

// verifySlackSignature verifies the Slack request signature.
func (h *SlackInteractionHandler) verifySlackSignature(header http.Header, body []byte) error {
	timestamp := header.Get("X-Slack-Request-Timestamp")
	signature := header.Get("X-Slack-Signature")

	if timestamp == "" || signature == "" {
		return fmt.Errorf("missing timestamp or signature headers")
	}

	// Check timestamp is recent (within 5 minutes)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	if abs(time.Now().Unix()-ts) > 60*5 {
		return fmt.Errorf("timestamp too old")
	}

	// Compute expected signature
	sigBaseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(h.signingSecret))
	mac.Write([]byte(sigBaseString))
	expectedSig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// SlackEventsHandler handles Slack Events API requests (URL verification, etc.).
type SlackEventsHandler struct {
	signingSecret string
	logger        alert.Logger
}

// NewSlackEventsHandler creates a new Slack events handler.
func NewSlackEventsHandler(signingSecret string, logger alert.Logger) *SlackEventsHandler {
	return &SlackEventsHandler{
		signingSecret: signingSecret,
		logger:        logger,
	}
}

// ServeHTTP handles POST /webhook/slack/events
func (h *SlackEventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Parse the event
	var event struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
	}

	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	// Handle URL verification challenge
	if event.Type == "url_verification" {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(event.Challenge))
		return
	}

	// For other events, acknowledge
	w.WriteHeader(http.StatusOK)
}

// abs returns the absolute value of an int64.
func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// parseMessageID parses a message ID from "channel:timestamp" format.
func parseMessageID(messageID string) (channelID, timestamp string, err error) {
	parts := strings.SplitN(messageID, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid message ID format: %s", messageID)
	}
	return parts[0], parts[1], nil
}
