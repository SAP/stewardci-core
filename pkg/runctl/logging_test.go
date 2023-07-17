package runctl

import (
	"context"
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/ktesting"
)

type runStatus struct {
	auxiliaryNamespace string
	namespace          string
	state              api.State
}

type runMeta struct {
	name      string
	namespace string
	UID       types.UID
}

func Test_logging_extendContextLoggerWithPipelineRunInfo(t *testing.T) {
	const logText = "This is a test"

	tests := []struct {
		name               string
		ctx                context.Context
		loggerName         string
		pipelineRun        k8s.PipelineRun
		additionalKVs      []interface{}
		expectedLoggerName string
		expectedLogKVs     []interface{}
	}{
		{
			name: "Background Context is provided",
			ctx:  context.Background(),
			pipelineRun: mockPipelineRun(
				runMeta{
					name:      "run1",
					namespace: "run-ns1",
					UID:       types.UID("123-abc"),
				},
				runStatus{
					namespace: "foo",
					state:     api.StateNew,
				},
			),
			additionalKVs:      nil,
			expectedLoggerName: "",
			expectedLogKVs: []interface{}{
				"pipelineRun", klog.ObjectRef{Name: "run1", Namespace: "run-ns1"},
				"pipelineRunUID", types.UID("123-abc"),
				"pipelineRunState", api.StateNew,
				"pipelineRunExecutionNamespace", "foo",
			},
		},
		{
			name:       "Logger name as a non-empty string",
			ctx:        context.Background(),
			loggerName: "tester",
			pipelineRun: mockPipelineRun(
				runMeta{
					name:      "run1",
					namespace: "run-ns1",
					UID:       types.UID("123-abc"),
				},
				runStatus{
					namespace: "foo",
					state:     api.StateNew,
				},
			),
			additionalKVs:      nil,
			expectedLoggerName: "tester",
			expectedLogKVs: []interface{}{
				"pipelineRun", klog.ObjectRef{Name: "run1", Namespace: "run-ns1"},
				"pipelineRunUID", types.UID("123-abc"),
				"pipelineRunState", api.StateNew,
				"pipelineRunExecutionNamespace", "foo",
			},
		},
		{
			name: "Additional logging key-values are provided in Context",
			ctx:  context.Background(),
			pipelineRun: mockPipelineRun(
				runMeta{
					name:      "run1",
					namespace: "run-ns1",
					UID:       types.UID("123-abc"),
				},
				runStatus{
					namespace: "foo",
					state:     api.StateNew,
				},
			),
			additionalKVs:      []interface{}{"pi", 3.14},
			expectedLoggerName: "",
			expectedLogKVs: []interface{}{
				"pi", 3.14,
				"pipelineRun", klog.ObjectRef{Name: "run1", Namespace: "run-ns1"},
				"pipelineRunUID", types.UID("123-abc"),
				"pipelineRunState", api.StateNew,
				"pipelineRunExecutionNamespace", "foo",
			},
		},
		{
			name: "Pipeline run with auxiliary namespace in the status",
			ctx:  context.Background(),
			pipelineRun: mockPipelineRun(
				runMeta{
					name:      "run1",
					namespace: "run-ns1",
					UID:       types.UID("123-abc"),
				},
				runStatus{
					auxiliaryNamespace: "foo-additional",
					namespace:          "foo",
					state:              api.StateNew,
				},
			),
			additionalKVs:      nil,
			expectedLoggerName: "",
			expectedLogKVs: []interface{}{
				"pipelineRun", klog.ObjectRef{Name: "run1", Namespace: "run-ns1"},
				"pipelineRunUID", types.UID("123-abc"),
				"pipelineRunState", api.StateNew,
				"pipelineRunExecutionNamespace", "foo",
				"pipelineRunExecutionAuxiliaryNamespace", "foo-additional",
			},
		},
		{
			name:               "Pipeline run as a nil object",
			ctx:                context.Background(),
			pipelineRun:        nil,
			additionalKVs:      []interface{}{"pi", 3.14},
			expectedLoggerName: "",
			expectedLogKVs: []interface{}{
				"pi", 3.14,
			},
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

			logger = klog.LoggerWithValues(logger, test.additionalKVs...)

			if test.loggerName != "" {
				logger = klog.LoggerWithName(logger, test.loggerName)
			}

			ctx := klog.NewContext(test.ctx, logger)

			// EXERCISE
			ctx, logger = extendContextLoggerWithPipelineRunInfo(
				ctx,
				test.pipelineRun,
			)

			// VERIFY logger
			logger.Info(logText)

			logsFromLogger := getLogEntries(t, logger)

			assert.Assert(t, logsFromLogger[0].Prefix == test.expectedLoggerName)
			assert.Assert(t, logsFromLogger[0].Message == logText)
			assert.DeepEqual(t, logsFromLogger[0].WithKVList, test.expectedLogKVs)

			// VERIFY ctx
			loggerFromCtx := klog.FromContext(ctx)
			loggerFromCtx.Info(logText)

			logsFromCtxLogger := getLogEntries(t, loggerFromCtx)

			assert.Assert(t, logsFromCtxLogger[0].Prefix == test.expectedLoggerName)
			assert.Assert(t, logsFromCtxLogger[0].Message == logText)
			assert.DeepEqual(t, logsFromCtxLogger[0].WithKVList, test.expectedLogKVs)
		})
	}
}

func Test_logging_extendContextLoggerWithPipelineRunInfo_with_nil_context(t *testing.T) {
	//SETUP
	pipelineRun := mockPipelineRun(
		runMeta{
			name:      "run1",
			namespace: "run-ns1",
			UID:       types.UID("123-abc"),
		},
		runStatus{
			auxiliaryNamespace: "foo-additional",
			namespace:          "foo",
			state:              api.StateNew,
		},
	)

	//EXERCISE
	ctx, logger := extendContextLoggerWithPipelineRunInfo(nil, pipelineRun)

	//VERIFY
	assert.Assert(t, ctx != nil)
	assert.Assert(t, logger != klog.Logger{})

	// for logger
	func(t *testing.T, logger logr.Logger) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatal("Logging should not panic")
			}
		}()

		logger.Info("this is a test")
	}(t, logger)

	// for logger from Context
	func(t *testing.T, logger logr.Logger) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatal("Logging should not panic")
			}
		}()

		logger.Info("this is a test")
	}(t, klog.FromContext(ctx))
}

func getLogEntries(t *testing.T, logger logr.Logger) ktesting.Log {
	t.Helper()

	underlyingLogger, ok := logger.GetSink().(ktesting.Underlier)
	if !ok {
		t.Fatalf("should have had ktesting LogSink, got %T", logger.GetSink())
	}
	return underlyingLogger.GetBuffer().Data()
}

func mockPipelineRun(runMeta runMeta, runStatus runStatus) k8s.PipelineRun {
	run := &api.PipelineRun{
		Status: api.PipelineStatus{
			AuxiliaryNamespace: runStatus.auxiliaryNamespace,
			Namespace:          runStatus.namespace,
			State:              api.State(runStatus.state),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      runMeta.name,
			Namespace: runMeta.namespace,
			UID:       runMeta.UID,
		},
	}
	pr, _ := k8s.NewPipelineRun(context.Background(), run, nil)
	return pr
}
