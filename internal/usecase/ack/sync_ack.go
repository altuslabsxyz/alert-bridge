package ack

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/entity"
	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/logger"
	"github.com/qj0r9j0vc2/alert-bridge/internal/domain/repository"
)

// SyncAckInput contains acknowledgment details from any source.
type SyncAckInput struct {
	AlertID   string
	Source    entity.AckSource
	UserID    string
	UserEmail string
	UserName  string
	Note      string
	Duration  *time.Duration
}

// SyncAckOutput contains the result of acknowledgment synchronization.
type SyncAckOutput struct {
	Alert      *entity.Alert
	AckEvent   *entity.AckEvent
	SyncedTo   []string // Names of systems that were updated
	SyncErrors []SyncError
}

// SyncError represents a sync failure to a specific system.
type SyncError struct {
	System string
	Error  error
}

// AckSyncer defines the contract for synchronizing acknowledgments to external systems.
type AckSyncer interface {
	// Acknowledge marks an alert as acknowledged in the target system.
	Acknowledge(ctx context.Context, alert *entity.Alert, ackEvent *entity.AckEvent) error

	// SupportsAck returns true if the target system supports acknowledgment.
	SupportsAck() bool

	// Name returns the syncer identifier.
	Name() string
}

// Logger is the unified logging interface from domain layer.
type Logger = logger.Logger

// SyncAckUseCase handles acknowledgment synchronization across systems.
type SyncAckUseCase struct {
	alertRepo    repository.AlertRepository
	ackEventRepo repository.AckEventRepository
	syncers      []AckSyncer
	logger       Logger
}

// NewSyncAckUseCase creates a new SyncAckUseCase with dependencies.
func NewSyncAckUseCase(
	alertRepo repository.AlertRepository,
	ackEventRepo repository.AckEventRepository,
	syncers []AckSyncer,
	logger Logger,
) *SyncAckUseCase {
	return &SyncAckUseCase{
		alertRepo:    alertRepo,
		ackEventRepo: ackEventRepo,
		syncers:      syncers,
		logger:       logger,
	}
}

// Execute processes an acknowledgment and syncs to all connected systems.
func (uc *SyncAckUseCase) Execute(ctx context.Context, input SyncAckInput) (*SyncAckOutput, error) {
	output := &SyncAckOutput{}

	// 1. Load the alert
	alert, err := uc.alertRepo.FindByID(ctx, input.AlertID)
	if err != nil {
		return nil, fmt.Errorf("finding alert: %w", err)
	}
	if alert == nil {
		return nil, entity.ErrAlertNotFound
	}

	// 2. Create ack event
	ackEvent := entity.NewAckEvent(
		input.AlertID,
		input.Source,
		input.UserID,
		input.UserEmail,
		input.UserName,
	)
	if input.Note != "" {
		ackEvent.WithNote(input.Note)
	}
	if input.Duration != nil {
		ackEvent.WithDuration(*input.Duration)
	}

	// 3. Save ack event (for audit trail)
	if err := uc.ackEventRepo.Save(ctx, ackEvent); err != nil {
		return nil, fmt.Errorf("saving ack event: %w", err)
	}

	output.AckEvent = ackEvent

	// 4. Update alert state
	err = alert.Acknowledge(input.UserEmail, time.Now().UTC())
	if err != nil {
		// If already acknowledged, continue to sync (idempotent behavior)
		if !errors.Is(err, entity.ErrAlertAlreadyAcked) && !errors.Is(err, entity.ErrAlertAlreadyResolved) {
			return nil, fmt.Errorf("acknowledging alert: %w", err)
		}
		uc.logger.Debug("alert already acked/resolved, continuing sync",
			"alertID", alert.ID,
			"state", alert.State,
		)
	}

	// 5. Persist alert state change
	if err := uc.alertRepo.Update(ctx, alert); err != nil {
		return nil, fmt.Errorf("updating alert: %w", err)
	}

	output.Alert = alert

	// 6. Sync to other systems (excluding source)
	uc.syncToExternalSystems(ctx, alert, ackEvent, input.Source, output)

	uc.logger.Info("ack synced",
		"alertID", alert.ID,
		"source", input.Source,
		"userEmail", input.UserEmail,
		"syncedTo", output.SyncedTo,
	)

	return output, nil
}

// syncToExternalSystems syncs the acknowledgment to all external systems except the source.
func (uc *SyncAckUseCase) syncToExternalSystems(
	ctx context.Context,
	alert *entity.Alert,
	ackEvent *entity.AckEvent,
	source entity.AckSource,
	output *SyncAckOutput,
) {
	for _, syncer := range uc.syncers {
		// Skip if syncer doesn't support ack
		if !syncer.SupportsAck() {
			continue
		}

		// Skip syncing back to the source system
		if syncer.Name() == string(source) {
			uc.logger.Debug("skipping sync to source",
				"source", source,
				"syncer", syncer.Name(),
			)
			continue
		}

		// Check if we should sync based on existing message/incident ID
		if !uc.shouldSync(alert, syncer.Name()) {
			uc.logger.Debug("skipping sync - no message ID",
				"alertID", alert.ID,
				"syncer", syncer.Name(),
			)
			continue
		}

		// Sync to this system
		if err := syncer.Acknowledge(ctx, alert, ackEvent); err != nil {
			uc.logger.Error("failed to sync ack",
				"syncer", syncer.Name(),
				"alertID", alert.ID,
				"error", err,
			)
			output.SyncErrors = append(output.SyncErrors, SyncError{
				System: syncer.Name(),
				Error:  err,
			})
			continue
		}

		output.SyncedTo = append(output.SyncedTo, syncer.Name())
		uc.logger.Info("ack synced to external system",
			"syncer", syncer.Name(),
			"alertID", alert.ID,
		)
	}
}

// shouldSync determines if we should sync to a specific system.
func (uc *SyncAckUseCase) shouldSync(alert *entity.Alert, syncerName string) bool {
	return alert.HasExternalReference(syncerName)
}

// AddSyncer adds a syncer to the use case.
func (uc *SyncAckUseCase) AddSyncer(syncer AckSyncer) {
	uc.syncers = append(uc.syncers, syncer)
}
