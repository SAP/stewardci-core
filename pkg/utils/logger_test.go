package utils

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/ktesting"
)

func Test_pipelineRun_NewLoggingContextWithValues(t *testing.T) {

	const logText = "This is a test"

	const key1 = "foo"
	const value1 = "foo_value"

	const key2 = "bar"
	const value2 = 1.212

	tests := []struct {
		name               string
		ctx                context.Context
		existingLoggerName string
		existingLoggerKVs  []interface{}
		newLoggerName      string
		newLoggerKVs       []interface{}
		expectedLoggerName string
		expectedKVs        []interface{}
	}{
		{
			name:               "Update contextual parameters of existing logger",
			ctx:                context.Background(),
			existingLoggerName: "base",
			existingLoggerKVs:  []interface{}{key1, value1},
			newLoggerName:      "",
			newLoggerKVs:       []interface{}{key2, value2},
			expectedLoggerName: "base",
			expectedKVs:        []interface{}{key1, value1, key2, value2},
		},
		{
			name:               "No contextual logging values for existing logger",
			ctx:                context.Background(),
			existingLoggerName: "base",
			existingLoggerKVs:  nil,
			newLoggerName:      "extended",
			newLoggerKVs:       []interface{}{key1, value1},
			expectedLoggerName: "base/extended",
			expectedKVs:        []interface{}{key1, value1},
		},
		{
			name:               "No contextual logging values for sub-logger",
			ctx:                context.Background(),
			existingLoggerName: "base",
			existingLoggerKVs:  []interface{}{key1, value1},
			newLoggerName:      "extended",
			newLoggerKVs:       nil,
			expectedLoggerName: "base/extended",
			expectedKVs:        []interface{}{key1, value1},
		},
		{
			name:               "No contextual logging values for existing and sub-logger",
			ctx:                nil,
			existingLoggerName: "base",
			existingLoggerKVs:  nil,
			newLoggerName:      "extended",
			newLoggerKVs:       nil,
			expectedLoggerName: "base/extended",
			expectedKVs:        nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// SETUP
			existingLogger := ktesting.NewLogger(
				t,
				ktesting.NewConfig(
					ktesting.BufferLogs(true),
				),
			)
			if test.existingLoggerName != "" {
				existingLogger = klog.LoggerWithName(existingLogger, test.existingLoggerName)
			}

			existingLogger = klog.LoggerWithValues(existingLogger, test.existingLoggerKVs...)

			// EXERCISE
			ctx := NewLoggingContextWithValues(
				test.ctx,
				&existingLogger,
				test.newLoggerName,
				test.newLoggerKVs...,
			)

			// VERIFY
			assert.Assert(t, ctx != nil)

			logger := klog.FromContext(ctx)

			// Add a log entry into a buffer
			logger.Error(fmt.Errorf("some error"), logText)

			underlyingLogger, ok := logger.GetSink().(ktesting.Underlier)
			if !ok {
				t.Fatalf("should have had ktesting LogSink, got %T", logger.GetSink())
			}

			logs := underlyingLogger.GetBuffer().Data()

			assert.Assert(t, logs[0].Message == logText)
			assert.Assert(t, logs[0].Prefix == test.expectedLoggerName)
			assert.DeepEqual(t, logs[0].WithKVList, test.expectedKVs)
		})
	}
}

func Test_pipelineRun_NewLoggingContextWithValues_nil_logger(t *testing.T) {
	tests := []struct {
		name          string
		newLoggerName string
		newLoggerKVs  []interface{}
	}{
		{
			name:          "Neither existing logger nor new logger but key values are provided",
			newLoggerName: "",
			newLoggerKVs:  []interface{}{"foo", 111},
		},
		{
			name:          "No existing logger but add sub-logger",
			newLoggerName: "sub",
			newLoggerKVs:  []interface{}{"foo", 111},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// EXERCISE
			ctx := NewLoggingContextWithValues(context.Background(),
				nil,
				test.newLoggerName,
				test.newLoggerKVs...,
			)

			// VERIFY
			assert.Assert(t, ctx != nil)

			logger := klog.FromContext(ctx)

			assert.Assert(t, logger != klog.Logger{})

			// checking for the panic while logging
			func(t *testing.T, logger logr.Logger) {
				defer func() {
					if r := recover(); r != nil {
						t.Fatal("Logging should not panic")
					}
				}()

				logger.Info("this is a test")
			}(t, logger)
		})
	}
}
