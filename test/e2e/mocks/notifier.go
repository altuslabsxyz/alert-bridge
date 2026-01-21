package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/entity"
)

// RecordedNotification represents a notification captured by the mock
type RecordedNotification struct {
	AlertID     string
	Fingerprint string
	Name        string
	Instance    string
	Severity    entity.AlertSeverity
	State       entity.AlertState
	Labels      map[string]string
	Annotations map[string]string
	MessageID   string
	Timestamp   time.Time
	IsUpdate    bool
}

// MockNotifier is an in-memory mock implementation of alert.Notifier
type MockNotifier struct {
	mu            sync.RWMutex
	name          string
	notifications []RecordedNotification
	messageIDSeq  int64
	failNext      bool
	failError     error
}

// NewMockNotifier creates a new mock notifier
func NewMockNotifier(name string) *MockNotifier {
	return &MockNotifier{
		name:          name,
		notifications: make([]RecordedNotification, 0),
		messageIDSeq:  1000,
	}
}

// Notify implements alert.Notifier
func (m *MockNotifier) Notify(ctx context.Context, alert *entity.Alert) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNext {
		m.failNext = false
		return "", m.failError
	}

	m.messageIDSeq++
	messageID := fmt.Sprintf("%s:%d", m.name, m.messageIDSeq)

	notification := RecordedNotification{
		AlertID:     alert.ID,
		Fingerprint: alert.Fingerprint,
		Name:        alert.Name,
		Instance:    alert.Instance,
		Severity:    alert.Severity,
		State:       alert.State,
		Labels:      copyMap(alert.Labels),
		Annotations: copyMap(alert.Annotations),
		MessageID:   messageID,
		Timestamp:   time.Now(),
		IsUpdate:    false,
	}

	m.notifications = append(m.notifications, notification)
	return messageID, nil
}

// UpdateMessage implements alert.Notifier
func (m *MockNotifier) UpdateMessage(ctx context.Context, messageID string, alert *entity.Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNext {
		m.failNext = false
		return m.failError
	}

	notification := RecordedNotification{
		AlertID:     alert.ID,
		Fingerprint: alert.Fingerprint,
		Name:        alert.Name,
		Instance:    alert.Instance,
		Severity:    alert.Severity,
		State:       alert.State,
		Labels:      copyMap(alert.Labels),
		Annotations: copyMap(alert.Annotations),
		MessageID:   messageID,
		Timestamp:   time.Now(),
		IsUpdate:    true,
	}

	m.notifications = append(m.notifications, notification)
	return nil
}

// Name implements alert.Notifier
func (m *MockNotifier) Name() string {
	return m.name
}

// Reset clears all recorded notifications
func (m *MockNotifier) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = make([]RecordedNotification, 0)
	m.failNext = false
	m.failError = nil
}

// GetNotifications returns all recorded notifications
func (m *MockNotifier) GetNotifications() []RecordedNotification {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]RecordedNotification, len(m.notifications))
	copy(result, m.notifications)
	return result
}

// GetNotificationsByFingerprint returns notifications matching the fingerprint
func (m *MockNotifier) GetNotificationsByFingerprint(fingerprint string) []RecordedNotification {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []RecordedNotification
	for i := range m.notifications {
		if m.notifications[i].Fingerprint == fingerprint {
			result = append(result, m.notifications[i])
		}
	}
	return result
}

// GetNotificationCount returns the total number of notifications
func (m *MockNotifier) GetNotificationCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.notifications)
}

// SetFailNext configures the mock to fail on the next call
func (m *MockNotifier) SetFailNext(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failNext = true
	m.failError = err
}

// HasNotificationWithFingerprint checks if a notification exists with the given fingerprint
func (m *MockNotifier) HasNotificationWithFingerprint(fingerprint string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := range m.notifications {
		if m.notifications[i].Fingerprint == fingerprint {
			return true
		}
	}
	return false
}

// GetLatestNotification returns the most recent notification (or nil if none)
func (m *MockNotifier) GetLatestNotification() *RecordedNotification {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.notifications) == 0 {
		return nil
	}
	latest := m.notifications[len(m.notifications)-1]
	return &latest
}

// copyMap creates a shallow copy of a string map
func copyMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
