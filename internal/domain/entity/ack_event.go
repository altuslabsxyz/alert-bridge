package entity

import (
	"time"

	"github.com/google/uuid"
)

// AckSource identifies where an acknowledgment originated.
type AckSource string

const (
	AckSourceSlack     AckSource = "slack"
	AckSourcePagerDuty AckSource = "pagerduty"
	AckSourceAPI       AckSource = "api"
)

// AckEvent represents an acknowledgment action on an alert.
// This is an immutable value object used for audit trail.
type AckEvent struct {
	// ID is the unique identifier for this ack event.
	ID string

	// AlertID references the alert that was acknowledged.
	AlertID string

	// Source identifies where the acknowledgment originated.
	Source AckSource

	// UserID is the platform-specific user identifier.
	UserID string

	// UserEmail is the normalized email for cross-platform correlation.
	UserEmail string

	// UserName is the display name of the user.
	UserName string

	// Note is an optional comment from the user.
	Note string

	// Duration is the silence/snooze duration if applicable.
	Duration *time.Duration

	// CreatedAt is when the ack event was created.
	CreatedAt time.Time
}

// NewAckEvent creates a new acknowledgment event.
func NewAckEvent(alertID string, source AckSource, userID, userEmail, userName string) *AckEvent {
	return &AckEvent{
		ID:        uuid.New().String(),
		AlertID:   alertID,
		Source:    source,
		UserID:    userID,
		UserEmail: userEmail,
		UserName:  userName,
		CreatedAt: time.Now().UTC(),
	}
}

// WithNote adds an optional note to the ack event and returns the event.
func (e *AckEvent) WithNote(note string) *AckEvent {
	e.Note = note
	return e
}

// WithDuration sets a silence/snooze duration and returns the event.
func (e *AckEvent) WithDuration(d time.Duration) *AckEvent {
	e.Duration = &d
	return e
}

// HasDuration returns true if a duration was specified.
func (e *AckEvent) HasDuration() bool {
	return e.Duration != nil && *e.Duration > 0
}

// HasNote returns true if a note was provided.
func (e *AckEvent) HasNote() bool {
	return e.Note != ""
}

// IsFromSlack returns true if the ack originated from Slack.
func (e *AckEvent) IsFromSlack() bool {
	return e.Source == AckSourceSlack
}

// IsFromPagerDuty returns true if the ack originated from PagerDuty.
func (e *AckEvent) IsFromPagerDuty() bool {
	return e.Source == AckSourcePagerDuty
}
