package errors

import (
	"fmt"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// WrapError is a wrapped k8s error
type WrapError interface {
	IsNotFound() bool
	Error() string
}

type wrapError struct {
	msg string
	err error
}

// Errorf returns new error with message defined by format and args
// If error is not nil, err.Error() is attached to the message
func Errorf(err error, format string, args ...interface{}) WrapError {
	message := fmt.Sprintf(format, args...)
	if err != nil {
		message = fmt.Sprintf("%s: %s", message, err.Error())
	}
	return &wrapError{
		msg: message,
		err: err,
	}
}

// IsNotFound returns true if error wraps an NotFound error
func (e *wrapError) IsNotFound() bool {
	return k8serrors.IsNotFound(e.err)
}

// Errors returns the error message
func (e *wrapError) Error() string {
	return e.msg
}
