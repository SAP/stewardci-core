package errors

import (
	"errors"
)

type recoverabilityAnnotation struct {
	wrapped     error
	recoverable bool
}

// let compiler verify interface compliance
var _ error = (*recoverabilityAnnotation)(nil)

func (a *recoverabilityAnnotation) Error() string {
	return a.wrapped.Error()
}

func (a *recoverabilityAnnotation) Unwrap() error {
	return a.wrapped
}

// errors.Is() would work without this method, but it
// provides a shortcut in case target is the wrapped error.
func (a *recoverabilityAnnotation) Is(target error) bool {
	return errors.Is(a.wrapped, target)
}

// Recoverable annotates a given error as recoverable.
// It is equivalent to `RecoverableIf(err, true)`
func Recoverable(err error) error {
	return RecoverableIf(err, true)
}

// NonRecoverable annotates a given error as non-recoverable.
// It is equivalent to `RecoverableIf(err, false)`
func NonRecoverable(err error) error {
	return RecoverableIf(err, false)
}

// RecoverableIf conditionally annotates a given error as recoverable.
// If err is nil, the function returns nil.
// If the recoverability status of err is equal to cond, the function
// returns err itself.
// Otherwise a new error is returned that wraps err.
func RecoverableIf(err error, cond bool) error {
	if err == nil {
		return nil
	}
	if IsRecoverable(err) == cond {
		// don't wrap if not necessary
		return err
	}
	return &recoverabilityAnnotation{
		wrapped:     err,
		recoverable: cond,
	}
}

// IsRecoverable returns true if the given error has been marked as
// recoverable.
func IsRecoverable(err error) bool {
	if err == nil {
		return false
	}
	if annotation := (*recoverabilityAnnotation)(nil); errors.As(err, &annotation) {
		return annotation.recoverable
	}
	return false
}
