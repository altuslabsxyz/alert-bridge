package dto

import (
	"time"
)

// SlackInteractionInput represents a Slack interactive component action.
type SlackInteractionInput struct {
	// ActionID is the action identifier (e.g., "ack_<alertID>").
	ActionID string

	// AlertID is the alert ID extracted from the action.
	AlertID string

	// UserID is the Slack user ID who performed the action.
	UserID string

	// UserName is the Slack username.
	UserName string

	// UserEmail is the user's email (if available).
	UserEmail string

	// ResponseURL is the URL to respond to the interaction.
	ResponseURL string

	// ChannelID is the channel where the interaction occurred.
	ChannelID string

	// MessageTS is the timestamp of the message.
	MessageTS string

	// Value is the action value (e.g., duration for silence).
	Value string

	// TriggerID is used for opening modals.
	TriggerID string
}

// SlackInteractionOutput represents the result of handling a Slack interaction.
type SlackInteractionOutput struct {
	// Success indicates if the interaction was handled successfully.
	Success bool

	// Message is an optional message to display.
	Message string

	// SilenceID is set if a silence was created.
	SilenceID string

	// SilenceEndAt is when the silence expires.
	SilenceEndAt *time.Time
}
