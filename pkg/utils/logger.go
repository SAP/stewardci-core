package utils

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
)

// LoggerFromContext returns logr.Logger implementation retrieved from
// ctx sent by caller. If no logger was set in ctx then a default
// logr.Logger implementation is returned.
func LoggerFromContext(ctx context.Context) logr.Logger {
	return klog.FromContext(ctx)
}

// LoggerWithName returns a new logger instance with name appended to
// logger.
func LoggerWithName(logger logr.Logger, name string) logr.Logger {
	return klog.LoggerWithName(logger, name)
}

// LoggerWithValues returns a new logger instance with key-value pairs (kvs)
func LoggerWithValues(logger logr.Logger, kvs ...interface{}) logr.Logger {
	return klog.LoggerWithValues(logger, kvs...)
}

// NewLoggingContextWithValues returns a new logger Context with provided key-value pairs via kvs.
// If non-empty `loggerName` is provided then name of `logger` will be appended with 'loggerName'.
//
// ctx is the existing Context.
//
// logger is a pointer to logr.Logger.
//
// loggerName is optional and indicates the suffix to the existing logger.
// The logging context (key-value pairs) of provided logger will be preserved in new logger logger instance.
// If provided logger has name "foo" and loggerName is "bar" then the extended logger name will be
// "foo/bar". loggerName as empty string will extend the key-value pairs in the Context provided by kvs.
//
// kvs is a slice with the elements as key-value pairs for structured logging.
// If nil or empty slice provided logger's Context will carried forward.
// For example - ["key1", "value1", "key2", "value2"]
func NewLoggingContextWithValues(ctx context.Context, logger *logr.Logger, loggerName string, kvs ...interface{}) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if logger == nil {
		l := LoggerFromContext(ctx)
		logger = &l
	}

	if loggerName != "" {
		*logger = LoggerWithName(*logger, loggerName)
	}

	if kvs != nil {
		*logger = LoggerWithValues(*logger, kvs...)
	}

	return klog.NewContext(ctx, *logger)
}
