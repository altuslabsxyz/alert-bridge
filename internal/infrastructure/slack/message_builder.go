package slack

import (
	"fmt"
	"strings"
	"time"

	"github.com/slack-go/slack"

	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/entity"
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
	var blocks []slack.Block

	// Header with status emoji and alert name
	headerText := b.buildHeader(alert)
	blocks = append(blocks, slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, headerText, true, false),
	))

	// Alert details section
	blocks = append(blocks, b.buildDetailsSection(alert))

	// Summary section
	if alert.Summary != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, alert.Summary, false, false),
			nil, nil,
		))
	}

	// Divider
	blocks = append(blocks, slack.NewDividerBlock())

	// Status context block
	blocks = append(blocks, b.buildStatusContext(alert))

	// Action buttons (only if alert is firing and not acknowledged)
	if alert.IsActive() {
		blocks = append(blocks, b.buildActionButtons(alert.ID))
	}

	return blocks
}

// buildHeader creates the header text with appropriate emoji.
func (b *MessageBuilder) buildHeader(alert *entity.Alert) string {
	var emoji string
	switch {
	case alert.IsResolved():
		emoji = "✅"
	case alert.IsAcked():
		emoji = "👀"
	case alert.Severity == entity.SeverityCritical:
		emoji = "🔴"
	case alert.Severity == entity.SeverityWarning:
		emoji = "🟡"
	default:
		emoji = "🔵"
	}

	return fmt.Sprintf("%s %s", emoji, alert.Name)
}

// buildDetailsSection creates the details section with fields.
func (b *MessageBuilder) buildDetailsSection(alert *entity.Alert) *slack.SectionBlock {
	fields := []*slack.TextBlockObject{
		slack.NewTextBlockObject(slack.MarkdownType,
			fmt.Sprintf("*Instance:*\n%s", b.valueOrNA(alert.Instance)), false, false),
		slack.NewTextBlockObject(slack.MarkdownType,
			fmt.Sprintf("*Severity:*\n%s", strings.Title(string(alert.Severity))), false, false),
	}

	if alert.Target != "" {
		fields = append(fields,
			slack.NewTextBlockObject(slack.MarkdownType,
				fmt.Sprintf("*Target:*\n%s", alert.Target), false, false))
	}

	// Add status field
	fields = append(fields,
		slack.NewTextBlockObject(slack.MarkdownType,
			fmt.Sprintf("*Status:*\n%s", b.formatState(alert.State)), false, false))

	return slack.NewSectionBlock(nil, fields, nil)
}

// buildStatusContext creates the status context block.
func (b *MessageBuilder) buildStatusContext(alert *entity.Alert) *slack.ContextBlock {
	var elements []slack.MixedElement

	// Fired time
	firedAt := alert.FiredAt.Format("Jan 2, 15:04 MST")
	elements = append(elements,
		slack.NewTextBlockObject(slack.MarkdownType,
			fmt.Sprintf("🔥 Fired: %s", firedAt), false, false))

	// Acknowledged info
	if alert.IsAcked() && alert.AckedBy != "" {
		ackedAt := "unknown"
		if alert.AckedAt != nil {
			ackedAt = alert.AckedAt.Format("15:04 MST")
		}
		elements = append(elements,
			slack.NewTextBlockObject(slack.MarkdownType,
				fmt.Sprintf("✅ Acked by %s at %s", alert.AckedBy, ackedAt), false, false))
	}

	// Resolved info
	if alert.IsResolved() && alert.ResolvedAt != nil {
		resolvedAt := alert.ResolvedAt.Format("15:04 MST")
		elements = append(elements,
			slack.NewTextBlockObject(slack.MarkdownType,
				fmt.Sprintf("🟢 Resolved: %s", resolvedAt), false, false))
	}

	return slack.NewContextBlock("", elements...)
}

// buildActionButtons creates the interactive action buttons.
func (b *MessageBuilder) buildActionButtons(alertID string) *slack.ActionBlock {
	elements := []slack.BlockElement{
		// Acknowledge button
		slack.NewButtonBlockElement(
			fmt.Sprintf("ack_%s", alertID),
			alertID,
			slack.NewTextBlockObject(slack.PlainTextType, "Acknowledge", true, false),
		).WithStyle(slack.StylePrimary),

		// Add Note button
		slack.NewButtonBlockElement(
			fmt.Sprintf("note_%s", alertID),
			alertID,
			slack.NewTextBlockObject(slack.PlainTextType, "Add Note", true, false),
		),
	}

	// Silence duration dropdown
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

	return slack.NewActionBlock(fmt.Sprintf("actions_%s", alertID), elements...)
}

// BuildAckedMessage creates a message for an acknowledged alert (buttons disabled).
func (b *MessageBuilder) BuildAckedMessage(alert *entity.Alert) []slack.Block {
	var blocks []slack.Block

	// Header
	headerText := b.buildHeader(alert)
	blocks = append(blocks, slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, headerText, true, false),
	))

	// Details
	blocks = append(blocks, b.buildDetailsSection(alert))

	// Summary
	if alert.Summary != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, alert.Summary, false, false),
			nil, nil,
		))
	}

	// Divider
	blocks = append(blocks, slack.NewDividerBlock())

	// Status context
	blocks = append(blocks, b.buildStatusContext(alert))

	return blocks
}

// formatDuration formats a duration for display.
func (b *MessageBuilder) formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
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

// valueOrNA returns the value or "N/A" if empty.
func (b *MessageBuilder) valueOrNA(value string) string {
	if value == "" {
		return "N/A"
	}
	return value
}
