package slack

import (
	"fmt"
	"strings"
	"time"

	"github.com/slack-go/slack"

	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/entity"
)

// Bright, modern color palette
const (
	colorCritical = "#FF6B6B" // Coral Red - bright & urgent
	colorWarning  = "#FFD93D" // Sunny Yellow - attention
	colorInfo     = "#6BCB77" // Mint Green - calm info
	colorResolved = "#4ECDC4" // Turquoise - fresh resolved
	colorAcked    = "#A78BFA" // Lavender - soft acknowledged
)

// MessageBuilder constructs Slack Block Kit messages for alerts.
type MessageBuilder struct {
	silenceDurations []time.Duration
}

// NewMessageBuilder creates a new message builder with the given silence durations.
func NewMessageBuilder(silenceDurations []time.Duration) *MessageBuilder {
	if len(silenceDurations) == 0 {
		silenceDurations = []time.Duration{
			15 * time.Minute,
			1 * time.Hour,
			4 * time.Hour,
			24 * time.Hour,
		}
	}
	return &MessageBuilder{
		silenceDurations: silenceDurations,
	}
}

// BuildAlertMessage creates a Block Kit message for an alert.
func (b *MessageBuilder) BuildAlertMessage(alert *entity.Alert) []slack.Block {
	return b.buildMessage(alert, true, true)
}

// BuildAckedMessage creates a message for an acknowledged alert with silence button still available.
func (b *MessageBuilder) BuildAckedMessage(alert *entity.Alert) []slack.Block {
	return b.buildMessage(alert, false, true)
}

// BuildResolvedMessage creates a message for a resolved alert (no buttons).
func (b *MessageBuilder) BuildResolvedMessage(alert *entity.Alert) []slack.Block {
	return b.buildMessage(alert, false, false)
}

// buildMessage creates a clean, bright Block Kit message.
func (b *MessageBuilder) buildMessage(alert *entity.Alert, showAckButton, showSilenceButton bool) []slack.Block {
	var blocks []slack.Block

	// Clean header with emoji + name
	emoji, _, _ := b.getStatusInfo(alert)
	headerText := fmt.Sprintf("%s  %s", emoji, alert.Name)
	blocks = append(blocks, slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, headerText, true, false),
	))

	// Summary (if available) - light and simple
	if alert.Summary != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, alert.Summary, false, false),
			nil, nil,
		))
	}

	// Compact info line
	blocks = append(blocks, b.buildCompactInfo(alert))

	// Action buttons (configurable)
	if showAckButton || showSilenceButton {
		if actionBlock := b.buildActionButtons(alert.ID, showAckButton, showSilenceButton); actionBlock != nil {
			blocks = append(blocks, actionBlock)
		}
	}

	// Subtle footer
	blocks = append(blocks, b.buildTimelineContext(alert))

	return blocks
}

// buildCompactInfo creates a single-line info display - clean and minimal.
func (b *MessageBuilder) buildCompactInfo(alert *entity.Alert) *slack.ContextBlock {
	var elements []slack.MixedElement

	// Status
	_, statusText, _ := b.getStatusInfo(alert)
	elements = append(elements,
		slack.NewTextBlockObject(slack.MarkdownType,
			fmt.Sprintf("*%s*", statusText), false, false))

	// Instance
	if alert.Instance != "" {
		elements = append(elements,
			slack.NewTextBlockObject(slack.MarkdownType,
				fmt.Sprintf("`%s`", alert.Instance), false, false))
	}

	// Target
	if alert.Target != "" {
		elements = append(elements,
			slack.NewTextBlockObject(slack.MarkdownType,
				fmt.Sprintf("`%s`", alert.Target), false, false))
	}

	return slack.NewContextBlock("", elements...)
}

// getStatusInfo returns emoji, text, and color for the alert status.
func (b *MessageBuilder) getStatusInfo(alert *entity.Alert) (emoji, text, color string) {
	switch {
	case alert.IsResolved():
		return "🟢", "Resolved", colorResolved
	case alert.IsAcked():
		return "👀", "Acknowledged", colorAcked
	case alert.Severity == entity.SeverityCritical:
		return "🔴", "Critical", colorCritical
	case alert.Severity == entity.SeverityWarning:
		return "🟡", "Warning", colorWarning
	default:
		return "🔵", "Info", colorInfo
	}
}

// getSeverityBadge returns a formatted severity badge.
func (b *MessageBuilder) getSeverityBadge(alert *entity.Alert) string {
	severity := strings.ToUpper(string(alert.Severity))
	return fmt.Sprintf("`%s`", severity)
}

// buildDetailsSection creates the details section with fields in a clean layout.
func (b *MessageBuilder) buildDetailsSection(alert *entity.Alert) *slack.SectionBlock {
	var fields []*slack.TextBlockObject

	// Instance
	if alert.Instance != "" {
		fields = append(fields,
			slack.NewTextBlockObject(slack.MarkdownType,
				fmt.Sprintf("*Instance*\n`%s`", alert.Instance), false, false))
	}

	// Target
	if alert.Target != "" {
		fields = append(fields,
			slack.NewTextBlockObject(slack.MarkdownType,
				fmt.Sprintf("*Target*\n`%s`", alert.Target), false, false))
	}

	// Fingerprint (shortened for display)
	if alert.Fingerprint != "" {
		fp := alert.Fingerprint
		if len(fp) > 12 {
			fp = fp[:12] + "..."
		}
		fields = append(fields,
			slack.NewTextBlockObject(slack.MarkdownType,
				fmt.Sprintf("*ID*\n`%s`", fp), false, false))
	}

	return slack.NewSectionBlock(nil, fields, nil)
}

// buildTimelineContext creates a clean, minimal footer.
func (b *MessageBuilder) buildTimelineContext(alert *entity.Alert) *slack.ContextBlock {
	var elements []slack.MixedElement

	// Fired time
	firedAt := alert.FiredAt.Format("Jan 2 • 15:04")
	elements = append(elements,
		slack.NewTextBlockObject(slack.MarkdownType, firedAt, false, false))

	// Acknowledged by
	if alert.IsAcked() && alert.AckedBy != "" {
		elements = append(elements,
			slack.NewTextBlockObject(slack.MarkdownType,
				fmt.Sprintf("by %s", alert.AckedBy), false, false))
	}

	return slack.NewContextBlock("", elements...)
}

// buildActionButtons creates action buttons.
func (b *MessageBuilder) buildActionButtons(alertID string, showAck, showSilence bool) *slack.ActionBlock {
	var elements []slack.BlockElement

	// Acknowledge button
	if showAck {
		ackBtn := slack.NewButtonBlockElement(
			fmt.Sprintf("ack_%s", alertID),
			alertID,
			slack.NewTextBlockObject(slack.PlainTextType, "Acknowledge", true, false),
		)
		elements = append(elements, ackBtn)
	}

	// Silence dropdown
	if showSilence {
		options := make([]*slack.OptionBlockObject, len(b.silenceDurations))
		for i, d := range b.silenceDurations {
			options[i] = slack.NewOptionBlockObject(
				d.String(),
				slack.NewTextBlockObject(slack.PlainTextType, b.formatDuration(d), false, false),
				nil,
			)
		}

		silenceSelect := slack.NewOptionsSelectBlockElement(
			slack.OptTypeStatic,
			slack.NewTextBlockObject(slack.PlainTextType, "Silence...", false, false),
			fmt.Sprintf("silence_%s", alertID),
			options...,
		)
		elements = append(elements, silenceSelect)
	}

	if len(elements) == 0 {
		return nil
	}

	return slack.NewActionBlock(fmt.Sprintf("actions_%s", alertID), elements...)
}

// formatDuration formats a duration for display.
func (b *MessageBuilder) formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d min", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

// formatState formats the alert state for display.
func (b *MessageBuilder) formatState(state entity.AlertState) string {
	switch state {
	case entity.StateActive:
		return "🔴 Firing"
	case entity.StateAcked:
		return "👀 Acknowledged"
	case entity.StateResolved:
		return "🟢 Resolved"
	default:
		return string(state)
	}
}
