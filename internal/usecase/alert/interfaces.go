package alert

import (
	"context"

	"github.com/altuslabsxyz/alert-bridge/internal/domain/entity"
	"github.com/altuslabsxyz/alert-bridge/internal/domain/logger"
)

// Notifier defines the contract for sending alert notifications.
// OCP: New notification channels implement this interface.
type Notifier interface {
	// Notify sends an alert to the notification channel.
	// Returns a channel-specific message ID for tracking.
	Notify(ctx context.Context, alert *entity.Alert) (messageID string, err error)

	// UpdateMessage updates an existing notification (e.g., after ack or resolve).
	UpdateMessage(ctx context.Context, messageID string, alert *entity.Alert) error

	// Name returns the notifier identifier (e.g., "slack", "pagerduty").
	Name() string
}

// Logger is the unified logging interface from domain layer.
type Logger = logger.Logger
