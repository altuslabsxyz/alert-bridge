package slack

import (
	"fmt"
	"strings"
	"time"

	"github.com/slack-go/slack"

	"github.com/altuslabsxyz/alert-bridge/internal/domain/entity"
)

// Bright, modern color palette
const (
	colorCritical = "#FF6B6B" // Coral Red - bright & urgent
	colorWarning  = "#FFD93D" // Sunny Yellow - attention
	colorInfo     = "#6BCB77" // Mint Green - calm info
	colorResolved = "#4ECDC4" // Turquoise - fresh resolved
	colorAcked    = "#A78BFA" // Lavender - soft acknowledged
)

// Slack date format tokens for automatic timezone/locale conversion.
// See: https://api.slack.com/reference/surfaces/formatting#date-formatting
const (
	SlackDateFull      = "{date} {time}"       // "January 21st, 2024 3:00 PM"
	SlackDateShort     = "{date_short} {time}" // "Jan 21, 2024 3:00 PM"
	SlackDateLong      = "{date_long} {time}"  // "Monday, January 21st, 2024 3:00 PM"
	SlackTimeOnly      = "{time}"              // "3:00 PM"
	SlackDateShortOnly = "{date_short}"        // "Jan 21, 2024"
)

// FormatSlackTime formats a time using Slack's date formatting syntax.
// This enables automatic timezone conversion and locale-based translation
// for each Slack user viewing the message.
//
// Example output for different users viewing the same timestamp:
//   - Seoul user (KST, Korean): "2024년 1월 22일 오전 9:00"
//   - LA user (PST, English): "Jan 21, 2024 4:00 PM"
//   - London user (GMT, English): "22 Jan 2024 00:00"
func FormatSlackTime(t time.Time, format string) string {
	unix := t.Unix()
	// Fallback text shown in contexts that don't support Slack formatting (e.g., email notifications)
	fallback := t.UTC().Format("2006-01-02 15:04 UTC")
	return fmt.Sprintf("<!date^%d^%s|%s>", unix, format, fallback)
}

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

// BuildUserMentions creates a formatted string of Slack user mentions.
// Example output: "<@U123> <@U456> <@U789>"
func BuildUserMentions(userIDs []string) string {
	if len(userIDs) == 0 {
		return ""
	}

	var mentions []string
	for _, id := range userIDs {
		mentions = append(mentions, fmt.Sprintf("<@%s>", id))
	}
	return strings.Join(mentions, " ")
}

// BuildAlertMessage creates a Block Kit message for an alert.
func (b *MessageBuilder) BuildAlertMessage(alert *entity.Alert) []slack.Block {
	return b.buildMessage(alert, true, true, nil)
}

// BuildAlertMessageWithMentions creates a Block Kit message for an alert
// with subscriber mentions at the top.
func (b *MessageBuilder) BuildAlertMessageWithMentions(alert *entity.Alert, slackUserIDs []string) []slack.Block {
	return b.buildMessage(alert, true, true, slackUserIDs)
}

// BuildAckedMessage creates a message for an acknowledged alert with silence button still available.
func (b *MessageBuilder) BuildAckedMessage(alert *entity.Alert) []slack.Block {
	return b.buildMessage(alert, false, true, nil)
}

// BuildResolvedMessage creates a message for a resolved alert (no buttons).
func (b *MessageBuilder) BuildResolvedMessage(alert *entity.Alert) []slack.Block {
	return b.buildMessage(alert, false, false, nil)
}

// buildMessage creates a clean, bright Block Kit message with configurable button options.
// slackUserIDs is optional - if provided, mentions will be added at the top of the message.
func (b *MessageBuilder) buildMessage(alert *entity.Alert, showAckButton, showSilenceButton bool, slackUserIDs []string) []slack.Block {
	var blocks []slack.Block

	// Add user mentions section at the top if any subscribers matched
	if len(slackUserIDs) > 0 {
		mentions := BuildUserMentions(slackUserIDs)
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType,
				fmt.Sprintf(":bell: *Alert Subscribers:* %s", mentions), false, false),
			nil, nil,
		))
	}

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
// Uses Slack's date formatting for automatic timezone/locale conversion.
func (b *MessageBuilder) buildTimelineContext(alert *entity.Alert) *slack.ContextBlock {
	var elements []slack.MixedElement

	// Fired time - uses Slack date formatting for automatic timezone conversion
	firedAt := FormatSlackTime(alert.FiredAt, SlackDateShort)
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
