package k8s

import (
	"context"
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/ktesting"
)

func Test_pipelineRun_NewPipelineRunLoggingContext(t *testing.T) {
	const pipelineRunName = "run1"
	const pipelineRunNamespace = "run-ns1"
	const pipelineRunUID = types.UID("123-abc")
	const loggerName = "fooLogger"
	const logText = "This is a test"

	expectedPipelineRunKVs := []interface{}{
		"pipelineRunObject", klog.ObjectRef{Name: "run1", Namespace: "run-ns1"},
		"pipelineRunUID", types.UID("123-abc"),
	}

	tests := []struct {
		name           string
		ctx            context.Context
		pipelineRun    PipelineRun
		additionalKVs  []interface{}
		expectedLogKVs []interface{}
	}{
		{
			name:           "Logger with additional (contextual) key values data provided",
			additionalKVs:  []interface{}{"foo", 123},
			pipelineRun:    mockPipelineRun(pipelineRunName, pipelineRunNamespace, pipelineRunUID),
			expectedLogKVs: append([]interface{}{"foo", 123}, expectedPipelineRunKVs...),
		},
		{
			name:           "Logger without additional additional key value (contextual) data",
			additionalKVs:  []interface{}{},
			pipelineRun:    mockPipelineRun(pipelineRunName, pipelineRunNamespace, pipelineRunUID),
			expectedLogKVs: expectedPipelineRunKVs,
		},
		{
			name:           "Nil pipeline run object provided",
			additionalKVs:  []interface{}{"foo", 123},
			pipelineRun:    nil,
			expectedLogKVs: []interface{}{"foo", 123},
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			// SETUP
			logger := ktesting.NewLogger(
				t,
				ktesting.NewConfig(
					ktesting.BufferLogs(true),
				),
			)

			logger = klog.LoggerWithName(logger, loggerName)
			logger = klog.LoggerWithValues(logger, test.additionalKVs...)

			// EXERCISE
			ctx := NewPipelineRunLoggingContext(&logger, test.pipelineRun)

			// VERIFY
			assert.Assert(t, ctx != nil)

			logger = klog.FromContext(ctx)

			// Add a log entry into a buffer
			logger.Info(logText)

			underlyingLogger, ok := logger.GetSink().(ktesting.Underlier)
			if !ok {
				t.Fatalf("should have had ktesting LogSink, got %T", logger.GetSink())
			}
			logs := underlyingLogger.GetBuffer().Data()

			assert.Assert(t, logs[0].Prefix == loggerName)
			assert.Assert(t, logs[0].Message == logText)
			assert.DeepEqual(t, logs[0].WithKVList, test.expectedLogKVs)
		})
	}
}

func Test_pipelineRun_NewPipelineRunLoggingContext_nil_logger(t *testing.T) {

	// SETUP
	pr := mockPipelineRun(
		"run1",
		"run-ns1",
		types.UID("123-abc"),
	)

	// EXERCISE
	ctx := NewPipelineRunLoggingContext(
		nil,
		pr,
	)

	// VERIFY
	assert.Assert(t, ctx != nil)

	logger, err := logr.FromContext(ctx)

	assert.NilError(t, err)
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
}

func mockPipelineRun(runName, runNamespace string, runUID types.UID) PipelineRun {
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
	pr, _ := NewPipelineRun(context.Background(), run, nil)
	return pr
}
