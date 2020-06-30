package runctl

import (
	"errors"
)

// Error is a runManager error
type Error struct {
	e           error
	recoverable bool
}

// RecoverabilityInfo is exposed by errors that can be converted to an RecoverabilityInfo object
type RecoverabilityInfo interface {
	IsRecoverable() bool
}

// NewRecoverabilityInfoError creates a new error implementing the RecoverabilityInfo interface
func NewRecoverabilityInfoError(err error, recoverable bool) *Error {
	return &Error{e: err,
		recoverable: recoverable,
	}
}

// Error returns error string of the provided error
func (err *Error) Error() string {
	return err.e.Error()
}

// IsRecoverable returns true if error can be recovered from
func (err *Error) IsRecoverable() bool {
	return err.recoverable
}

// IsRecoverable returns true if error can be recovered from
func IsRecoverable(err error) bool {
	if recoverability := RecoverabilityInfo(nil); errors.As(err, &recoverability) {
		return recoverability.IsRecoverable()
	}
	return false
}
