package entity

import (
	"time"

	"github.com/google/uuid"
)

// AlertSeverity represents the urgency level of an alert.
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityWarning  AlertSeverity = "warning"
	SeverityInfo     AlertSeverity = "info"
)

// AlertState represents the current lifecycle state of an alert.
type AlertState string

const (
	StateActive   AlertState = "active"
	StateAcked    AlertState = "acknowledged"
	StateResolved AlertState = "resolved"
)

// Alert represents a monitored event that requires attention.
// This is the core domain entity - pure business logic, no infrastructure dependencies.
type Alert struct {
	// ID is the unique identifier for this alert instance.
	ID string

	// Fingerprint is the Alertmanager fingerprint for deduplication.
	Fingerprint string

	// Name is the human-readable alert name.
	Name string

	// Instance is the source that generated the alert (e.g., server name, pod ID).
	Instance string

	// Target is the monitored target (e.g., endpoint URL, service name).
	Target string

	// Summary is a brief description of the alert condition.
	Summary string

	// Description provides detailed context about the alert.
	Description string

	// Severity indicates the urgency level.
	Severity AlertSeverity

	// State is the current lifecycle state.
	State AlertState

	// Labels are key-value pairs from the alerting rule.
	Labels map[string]string

	// Annotations provide additional contextual information.
	Annotations map[string]string

	// ExternalReferences stores integration-specific message/incident IDs.
	// Keys: "slack", "pagerduty", "discord", etc.
	ExternalReferences map[string]string

	// FiredAt is when the alert first fired.
	FiredAt time.Time

	// AckedAt is when the alert was acknowledged.
	AckedAt *time.Time

	// AckedBy identifies who acknowledged the alert.
	AckedBy string

	// ResolvedAt is when the alert was resolved.
	ResolvedAt *time.Time

	// ResolvedBy identifies who manually resolved the alert (from Slack).
	ResolvedBy string

	// CreatedAt is when this record was created.
	CreatedAt time.Time

	// UpdatedAt is when this record was last updated.
	UpdatedAt time.Time
}

// NewAlert creates a new Alert with the given parameters.
func NewAlert(fingerprint, name, instance, target, summary string, severity AlertSeverity) *Alert {
	now := time.Now().UTC()
	return &Alert{
		ID:                 uuid.New().String(),
		Fingerprint:        fingerprint,
		Name:               name,
		Instance:           instance,
		Target:             target,
		Summary:            summary,
		Severity:           severity,
		State:              StateActive,
		Labels:             make(map[string]string),
		Annotations:        make(map[string]string),
		ExternalReferences: make(map[string]string),
		FiredAt:            now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// Acknowledge marks the alert as acknowledged.
// Returns ErrAlertAlreadyResolved if the alert is already resolved.
// Returns ErrAlertAlreadyAcked if the alert is already acknowledged.
func (a *Alert) Acknowledge(by string, at time.Time) error {
	if a.State == StateResolved {
		return ErrAlertAlreadyResolved
	}
	if a.State == StateAcked {
		return ErrAlertAlreadyAcked
	}

	a.State = StateAcked
	a.AckedAt = &at
	a.AckedBy = by
	a.UpdatedAt = at
	return nil
}

// Resolve marks the alert as resolved.
func (a *Alert) Resolve(at time.Time) {
	a.State = StateResolved
	a.ResolvedAt = &at
	a.UpdatedAt = at
}

// ResolveBy marks the alert as manually resolved by a specific user.
func (a *Alert) ResolveBy(by string, at time.Time) {
	a.State = StateResolved
	a.ResolvedAt = &at
	a.ResolvedBy = by
	a.UpdatedAt = at
}

// IsActive returns true if the alert is in active state.
func (a *Alert) IsActive() bool {
	return a.State == StateActive
}

// IsAcked returns true if the alert has been acknowledged.
func (a *Alert) IsAcked() bool {
	return a.State == StateAcked
}

// IsFiring returns true if the alert is not resolved (active or acknowledged).
func (a *Alert) IsFiring() bool {
	return a.State != StateResolved
}

// IsResolved returns true if the alert has been resolved.
func (a *Alert) IsResolved() bool {
	return a.State == StateResolved
}

// AddLabel adds a label to the alert.
func (a *Alert) AddLabel(key, value string) {
	if a.Labels == nil {
		a.Labels = make(map[string]string)
	}
	a.Labels[key] = value
}

// AddAnnotation adds an annotation to the alert.
func (a *Alert) AddAnnotation(key, value string) {
	if a.Annotations == nil {
		a.Annotations = make(map[string]string)
	}
	a.Annotations[key] = value
}

// SetExternalReference sets an external system reference ID.
func (a *Alert) SetExternalReference(system, referenceID string) {
	if a.ExternalReferences == nil {
		a.ExternalReferences = make(map[string]string)
	}
	a.ExternalReferences[system] = referenceID
	a.UpdatedAt = time.Now().UTC()
}

// GetExternalReference returns the external reference ID for a system.
func (a *Alert) GetExternalReference(system string) string {
	if a.ExternalReferences == nil {
		return ""
	}
	return a.ExternalReferences[system]
}

// HasExternalReference checks if an external reference exists for a system.
func (a *Alert) HasExternalReference(system string) bool {
	return a.GetExternalReference(system) != ""
}

// GetLabel returns the value of a label, or empty string if not found.
func (a *Alert) GetLabel(key string) string {
	if a.Labels == nil {
		return ""
	}
	return a.Labels[key]
}

// GetAnnotation returns the value of an annotation, or empty string if not found.
func (a *Alert) GetAnnotation(key string) string {
	if a.Annotations == nil {
		return ""
	}
	return a.Annotations[key]
}
