package k8s

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"strings"
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

var flagsInitialized bool

func klogInitFlags() {
	if !flagsInitialized {
		klog.InitFlags(nil)
		flagsInitialized = true
	}
}

func Test_pipelineRun_NewPipelineRunLoggingContext(t *testing.T) {
	pipelineRunName := "run1"
	pipelineRunNamespace := "ns1"
	pipelineRunUID := types.UID("123")

	klogInitFlags()

	tests := []struct {
		name       string
		ctx        context.Context
		loggerName string
	}{
		{
			name:       "No logger name and context provided for new logger",
			ctx:        nil,
			loggerName: "",
		},
		{
			name:       "No context provided for new logger",
			ctx:        nil,
			loggerName: "rootLogger",
		},
		{
			name:       "No logger name provided for new logger",
			ctx:        context.TODO(),
			loggerName: "",
		},
		{
			name:       "Valid logger name and context provided for new logger",
			ctx:        context.TODO(),
			loggerName: "rootLogger",
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			setCommandLineFlags()

			buffer := new(bytes.Buffer)
			klog.SetOutput(buffer)

			mockPipelineRun, err := mockPipelineRun(
				context.Background(),
				pipelineRunName,
				pipelineRunNamespace,
				pipelineRunUID,
			)
			assert.NilError(t, err)

			ctx := NewPipelineRunLoggingContext(test.ctx, test.loggerName, mockPipelineRun)
			assert.Assert(t, ctx != nil)

			logger := klog.FromContext(ctx)

			// Following logging will add a log line/stream into a buffer as an output
			// set for logger to verify the result.
			logPrefix := "This is a test"
			logger.Info(logPrefix)
			klog.Flush()

			kvFromLog := getContextKeyValuesSectionFromLogText(
				buffer.String(),
				fmt.Sprintf("%s\"", logPrefix),
			)

			// adjust the expected string if the contextual information of pipeline run is updated
			assert.Equal(t, kvFromLog, "pipelineRunObject=\"ns1/run1\" pipelineRunUID=123")
		})
	}
}
func Test_pipelineRun_UpdateLoggerContext(t *testing.T) {

	key1 := "foo"
	value1 := "foo_value"

	key2 := "bar"
	value2 := 1.212

	klogInitFlags()

	tests := []struct {
		name                  string
		existingCtx           context.Context
		existingLoggerName    string
		newLoggerName         string
		newLoggerKVs          []interface{}
		expectedLoggerName    string
		expectedKeyValuesText string
	}{
		{
			name:                  "Update context of existing logger",
			existingCtx:           context.Background(),
			existingLoggerName:    "base",
			newLoggerName:         "",
			newLoggerKVs:          []interface{}{key2, value2},
			expectedLoggerName:    "base",
			expectedKeyValuesText: "foo=\"foo_value\" bar=1.212",
		},
		{
			name:                  "Add new logger in existing logger",
			existingCtx:           context.Background(),
			existingLoggerName:    "base",
			newLoggerName:         "extended",
			newLoggerKVs:          []interface{}{key2, value2},
			expectedLoggerName:    "base/extended",
			expectedKeyValuesText: "foo=\"foo_value\" bar=1.212",
		},
		{
			name:                  "No existing logger but add new logger",
			existingCtx:           nil,
			existingLoggerName:    "",
			newLoggerName:         "extended",
			newLoggerKVs:          []interface{}{key2, value2},
			expectedLoggerName:    "extended",
			expectedKeyValuesText: "bar=1.212",
		},
		{
			name:                  "No new key value pairs for new logger",
			existingCtx:           context.Background(),
			existingLoggerName:    "base",
			newLoggerName:         "extended",
			newLoggerKVs:          []interface{}{},
			expectedLoggerName:    "base/extended",
			expectedKeyValuesText: "foo=\"foo_value\"",
		},
		{
			name:                  "Neither existing logger nor new logger but key values are provided",
			existingCtx:           nil,
			existingLoggerName:    "",
			newLoggerName:         "",
			newLoggerKVs:          []interface{}{key2, value2},
			expectedLoggerName:    "",
			expectedKeyValuesText: "bar=1.212",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setCommandLineFlags()

			buffer := new(bytes.Buffer)
			klog.SetOutput(buffer)

			var ctx context.Context
			var logger klog.Logger

			// prepare existing logger context
			if test.existingLoggerName != "" && test.existingCtx != nil {
				ctx = test.existingCtx
				logger = klog.LoggerWithValues(
					klog.LoggerWithName(klog.FromContext(ctx), test.existingLoggerName),
					key1, value1,
				)
				ctx = klog.NewContext(ctx, logger)
			}

			updatedCtx := UpdateLoggerContext(ctx, test.newLoggerName, test.newLoggerKVs...)

			assert.Assert(t, updatedCtx != nil)

			logger = klog.FromContext(updatedCtx)

			// Following logging will add a log line/stream into a buffer as an output
			// set for logger to verify the result.
			logPrefix := "This is a test"
			logger.Info(logPrefix)
			klog.Flush()

			loggerFullName := extractLoggerFullName(test.existingLoggerName, test.newLoggerName)

			kvFromLog := getContextKeyValuesSectionFromLogText(
				buffer.String(),
				fmt.Sprintf("%s\"", logPrefix),
			)

			assert.Equal(t, loggerFullName, test.expectedLoggerName)
			assert.Equal(t, kvFromLog, test.expectedKeyValuesText)
		})
	}
}

func mockPipelineRun(ctx context.Context, runName, runNamespace string, runUID types.UID) (PipelineRun, error) {
	run := &api.PipelineRun{
		Status: api.PipelineStatus{
			Namespace: "foo",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName,
			Namespace: runNamespace,
			UID:       runUID,
		},
	}
	return NewPipelineRun(ctx, run, nil)
}

func getContextKeyValuesSectionFromLogText(logMsg, splitDelimiter string) string {
	log := strings.SplitAfterN(logMsg, splitDelimiter, 2)
	return strings.TrimSuffix(strings.TrimSpace(log[1]), "\n")
}

func extractLoggerFullName(existingLoggerName, extendedLoggerName string) string {
	var loggerFullName string
	if existingLoggerName != "" && extendedLoggerName != "" {
		loggerFullName = fmt.Sprintf("%s/%s", existingLoggerName, extendedLoggerName)
	} else if existingLoggerName != "" {
		loggerFullName = existingLoggerName
	} else if extendedLoggerName != "" {
		loggerFullName = extendedLoggerName
	} else {
		loggerFullName = ""
	}
	return loggerFullName
}

func setCommandLineFlags() {
	flag.Set("logtostderr", "false")
	flag.Set("alsologtostderr", "false")
	flag.Parse()
}
