package errors

import (
	"errors"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
)

type errorClassAnnotation struct {
	wrapped error
	class   api.Result
}

// let compiler verify interface compliance
var _ error = (*errorClassAnnotation)(nil)

func (a *errorClassAnnotation) Error() string {
	return a.wrapped.Error()
}

func (a *errorClassAnnotation) Unwrap() error {
	return a.wrapped
}

// errors.Is() would work without this method, but it
// provides a shortcut in case target is the wrapped error.
func (a *errorClassAnnotation) Is(target error) bool {
	return errors.Is(a.wrapped, target)
}

// Classify annotates a given error with a error class.
func Classify(err error, class api.Result) error {
	return &errorClassAnnotation{
		wrapped: err,
		class:   class,
	}
}

// GetClass returns the class of the error.
func GetClass(err error) api.Result {
	if err == nil {
		return api.ResultUndefined
	}
	if annotation := (*errorClassAnnotation)(nil); errors.As(err, &annotation) {
		return annotation.class
	}
	return api.ResultUndefined
}
