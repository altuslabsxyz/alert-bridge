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

	mentions := make([]string, len(userIDs))
	for i, id := range userIDs {
		mentions[i] = fmt.Sprintf("<@%s>", id)
	}
	return strings.Join(mentions, " ")
}

// BuildAlertMessage creates a Block Kit message for an alert.
func (b *MessageBuilder) BuildAlertMessage(alert *entity.Alert) []slack.Block {
	return b.buildMessage(alert, true, true, false, nil)
}

// BuildAlertMessageWithMentions creates a Block Kit message for an alert
// with subscriber mentions at the top.
func (b *MessageBuilder) BuildAlertMessageWithMentions(alert *entity.Alert, slackUserIDs []string) []slack.Block {
	return b.buildMessage(alert, true, true, false, slackUserIDs)
}

// BuildAckedMessage creates a message for an acknowledged alert with Resolve button.
func (b *MessageBuilder) BuildAckedMessage(alert *entity.Alert) []slack.Block {
	return b.buildMessage(alert, false, true, true, nil)
}

// BuildResolvedMessage creates a message for a resolved alert (no buttons).
func (b *MessageBuilder) BuildResolvedMessage(alert *entity.Alert) []slack.Block {
	return b.buildMessage(alert, false, false, false, nil)
}

// buildMessage creates a clean, structured Block Kit message with configurable button options.
// The layout follows:
// - Status header with emoji and severity (e.g., "🟢 RESOLVED | WARNING")
// - Alert name
// - Summary/description
// - Details fields (Instance, Target, ID)
// - Action buttons
// - Timeline footer
// slackUserIDs is optional - if provided, mentions will be added at the top of the message.
func (b *MessageBuilder) buildMessage(alert *entity.Alert, showAckButton, showSilenceButton, showResolveButton bool, slackUserIDs []string) []slack.Block {
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

	// Status header with emoji and severity badge
	// Format: "🟢 RESOLVED | WARNING" or "🔴 FIRING | CRITICAL"
	emoji, statusText, _ := b.getStatusInfo(alert)
	severityBadge := strings.ToUpper(string(alert.Severity))
	statusHeader := fmt.Sprintf("%s *%s* | %s", emoji, statusText, severityBadge)
	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, statusHeader, false, false),
		nil, nil,
	))

	// Alert name as header
	blocks = append(blocks, slack.NewHeaderBlock(
		slack.NewTextBlockObject(slack.PlainTextType, alert.Name, true, false),
	))

	// Summary/description (if available)
	if alert.Summary != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, alert.Summary, false, false),
			nil, nil,
		))
	}

	// Details fields section
	blocks = append(blocks, b.buildDetailsFields(alert))

	// Action buttons (configurable)
	if showAckButton || showSilenceButton || showResolveButton {
		if actionBlock := b.buildActionButtons(alert.ID, showAckButton, showSilenceButton, showResolveButton); actionBlock != nil {
			blocks = append(blocks, actionBlock)
		}
	}

	// Timeline footer
	blocks = append(blocks, b.buildTimelineContext(alert))

	return blocks
}

// buildDetailsFields creates field blocks for Instance, Target, and ID.
func (b *MessageBuilder) buildDetailsFields(alert *entity.Alert) *slack.SectionBlock {
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

	// Fingerprint/ID (shortened for display)
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

// buildTimelineContext creates a clean, minimal footer with timeline information.
// Uses Slack's date formatting for automatic timezone/locale conversion.
// Format: "Fired: Jan 22, 10:31 • Acked: 10:45 by user • Resolved: 10:51 by user"
func (b *MessageBuilder) buildTimelineContext(alert *entity.Alert) *slack.ContextBlock {
	var parts []string

	// Fired time - uses Slack date formatting for automatic timezone conversion
	firedAt := FormatSlackTime(alert.FiredAt, SlackDateShort)
	parts = append(parts, fmt.Sprintf("*Fired:* %s", firedAt))

	// Acknowledged info
	if alert.IsAcked() && alert.AckedAt != nil {
		ackedAt := FormatSlackTime(*alert.AckedAt, SlackTimeOnly)
		if alert.AckedBy != "" {
			parts = append(parts, fmt.Sprintf("*Acked:* %s by %s", ackedAt, alert.AckedBy))
		} else {
			parts = append(parts, fmt.Sprintf("*Acked:* %s", ackedAt))
		}
	}

	// Resolved info
	if alert.IsResolved() && alert.ResolvedAt != nil {
		resolvedAt := FormatSlackTime(*alert.ResolvedAt, SlackTimeOnly)
		if alert.ResolvedBy != "" {
			parts = append(parts, fmt.Sprintf("*Resolved:* %s by %s", resolvedAt, alert.ResolvedBy))
		} else {
			parts = append(parts, fmt.Sprintf("*Resolved:* %s", resolvedAt))
		}
	}

	// Join with bullet separator
	timelineText := strings.Join(parts, " • ")

	return slack.NewContextBlock("",
		slack.NewTextBlockObject(slack.MarkdownType, timelineText, false, false))
}

// buildActionButtons creates action buttons.
func (b *MessageBuilder) buildActionButtons(alertID string, showAck, showSilence, showResolve bool) *slack.ActionBlock {
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

	// Resolve button (shown after ack, replaces ack button)
	if showResolve {
		resolveBtn := slack.NewButtonBlockElement(
			fmt.Sprintf("resolve_%s", alertID),
			alertID,
			slack.NewTextBlockObject(slack.PlainTextType, "Resolve", true, false),
		)
		resolveBtn.Style = slack.StylePrimary
		elements = append(elements, resolveBtn)
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
