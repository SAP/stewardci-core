package runctl

import (
	"errors"
)

type recoverableInfoError struct {
	err         error
	recoverable bool
}

// RecoverabilityInfo is exposed by errors that can be converted to an RecoverabilityInfo object
type RecoverabilityInfo interface {
        error 
	IsRecoverable() bool
}

// NewRecoverabilityInfoError creates a new error implementing the RecoverabilityInfo interface
func NewRecoverabilityInfoError(err error, recoverable bool) RecoverabilityInfo {
	if err == nil {
		panic("Cannot use nil as error")
	}

	return &recoverableInfoError{err: err,
		recoverable: recoverable,
	}
}

// Error returns error string of the provided error
func (err *recoverableInfoError) Error() string {
	return err.err.Error()
}

// IsRecoverable returns true if error can be recovered from
func (err *recoverableInfoError) IsRecoverable() bool {
	if err == nil {
		return false
	}
	return err.recoverable
}

// IsRecoverable returns true if error can be recovered from
func IsRecoverable(err error) bool {
	if err == nil {
		return false
	}
	if recoverability := RecoverabilityInfo(nil); errors.As(err, &recoverability) {
		return recoverability.IsRecoverable()
	}
	return false
}
