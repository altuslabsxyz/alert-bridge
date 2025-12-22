package entity

import "errors"

// Domain errors - sentinel errors for business logic validation.
var (
	// ErrAlertNotFound indicates the requested alert does not exist.
	ErrAlertNotFound = errors.New("alert not found")

	// ErrAlertAlreadyResolved indicates an action was attempted on a resolved alert.
	ErrAlertAlreadyResolved = errors.New("alert already resolved")

	// ErrAlertAlreadyAcked indicates the alert was already acknowledged.
	ErrAlertAlreadyAcked = errors.New("alert already acknowledged")

	// ErrInvalidAlertState indicates an invalid state transition was attempted.
	ErrInvalidAlertState = errors.New("invalid alert state transition")

	// ErrDuplicateAlert indicates an alert with the same ID already exists.
	ErrDuplicateAlert = errors.New("duplicate alert")

	// ErrSilenceNotFound indicates the requested silence does not exist.
	ErrSilenceNotFound = errors.New("silence not found")

	// ErrSilenceExpired indicates the silence has already expired.
	ErrSilenceExpired = errors.New("silence expired")

	// ErrInvalidSilenceDuration indicates an invalid silence duration was provided.
	ErrInvalidSilenceDuration = errors.New("invalid silence duration")
)

// IsNotFound checks if the error indicates a not-found condition.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrAlertNotFound) || errors.Is(err, ErrSilenceNotFound)
}

// IsConflict checks if the error indicates a conflict condition.
func IsConflict(err error) bool {
	return errors.Is(err, ErrAlertAlreadyResolved) ||
		errors.Is(err, ErrAlertAlreadyAcked) ||
		errors.Is(err, ErrDuplicateAlert)
}
