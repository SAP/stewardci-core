package log

import (
	"context"
	"testing"

	logrtesting "github.com/SAP/stewardci-core/internal/logr/testing"
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"github.com/SAP/stewardci-core/pkg/runctl/cfg"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sapitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	_ "knative.dev/pkg/system/testing"
)

func Test_extendLoggerWithPipelineRunInfo(t *testing.T) {
	tests := []struct {
		name            string
		pipelineRun     *api.PipelineRun
		expectedWithKVs []interface{}
	}{
		{
			name: "empty pipeline run object",

			pipelineRun: &api.PipelineRun{},
			expectedWithKVs: []interface{}{
				"pipelineRun", klog.ObjectRef{},
				"pipelineRunUID", k8sapitypes.UID(""),
				"pipelineRunState", api.StateUndefined,
			},
		},
		{
			name: "metadata.name",

			pipelineRun: &api.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name: "run-1",
				},
			},
			expectedWithKVs: []interface{}{
				"pipelineRun", klog.ObjectRef{Name: "run-1"},
				"pipelineRunUID", k8sapitypes.UID(""),
				"pipelineRunState", api.StateUndefined,
			},
		},
		{
			name: "metadata.namespace",

			pipelineRun: &api.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "run-namespace-1",
				},
			},
			expectedWithKVs: []interface{}{
				"pipelineRun", klog.ObjectRef{Namespace: "run-namespace-1"},
				"pipelineRunUID", k8sapitypes.UID(""),
				"pipelineRunState", api.StateUndefined,
			},
		},
		{
			name: "metadata.uid",

			pipelineRun: &api.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					UID: k8sapitypes.UID("uid-1"),
				},
			},
			expectedWithKVs: []interface{}{
				"pipelineRun", klog.ObjectRef{},
				"pipelineRunUID", k8sapitypes.UID("uid-1"),
				"pipelineRunState", api.StateUndefined,
			},
		},
		{
			name: "status.state",

			pipelineRun: &api.PipelineRun{
				Status: api.PipelineStatus{
					State: api.StateCleaning,
				},
			},
			expectedWithKVs: []interface{}{
				"pipelineRun", klog.ObjectRef{},
				"pipelineRunUID", k8sapitypes.UID(""),
				"pipelineRunState", api.StateCleaning,
			},
		},
		{
			name: "status.namespace",

			pipelineRun: &api.PipelineRun{
				Status: api.PipelineStatus{
					Namespace: "exec-namespace-1",
				},
			},
			expectedWithKVs: []interface{}{
				"pipelineRun", klog.ObjectRef{},
				"pipelineRunUID", k8sapitypes.UID(""),
				"pipelineRunState", api.StateUndefined,
				"pipelineRunExecNamespace", "exec-namespace-1",
			},
		},
		{
			name: "status.auxiliaryNamespace",

			pipelineRun: &api.PipelineRun{
				Status: api.PipelineStatus{
					AuxiliaryNamespace: "exec-aux-namespace-1",
				},
			},
			expectedWithKVs: []interface{}{
				"pipelineRun", klog.ObjectRef{},
				"pipelineRunUID", k8sapitypes.UID(""),
				"pipelineRunState", api.StateUndefined,
				"pipelineRunExecAuxNamespace", "exec-aux-namespace-1",
			},
		},
		{
			name: "all together",

			pipelineRun: &api.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "run-2",
					Namespace: "run-namespace-2",
					UID:       k8sapitypes.UID("uid-2"),
				},
				Status: api.PipelineStatus{
					State:              api.StatePreparing,
					Namespace:          "exec-namespace-2",
					AuxiliaryNamespace: "exec-aux-namespace-2",
				},
			},
			expectedWithKVs: []interface{}{
				"pipelineRun", klog.ObjectRef{Name: "run-2", Namespace: "run-namespace-2"},
				"pipelineRunUID", k8sapitypes.UID("uid-2"),
				"pipelineRunState", api.StatePreparing,
				"pipelineRunExecNamespace", "exec-namespace-2",
				"pipelineRunExecAuxNamespace", "exec-aux-namespace-2",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// SETUP
			g := NewGomegaWithT(t)
			mockCtrl := gomock.NewController(t)

			origSink := logrtesting.NewMockLogSink(mockCtrl)
			newSink := logrtesting.NewMockLogSink(mockCtrl)

			gomock.InOrder(
				origSink.EXPECT().Init(gomock.Any()),
				origSink.EXPECT().WithValues(gomock.Any()).DoAndReturn(
					func(args ...interface{}) logr.LogSink {
						g.Expect(args).To(HaveExactElements(test.expectedWithKVs...))
						return newSink
					},
				),
			)

			logger := logr.New(origSink)

			// EXERCISE
			resultLogger := extendLoggerWithPipelineRunInfo(
				logger,
				test.pipelineRun,
				nil,
			)

			// VERIFY
			g.Expect(resultLogger).NotTo(BeIdenticalTo(logger))
			g.Expect(resultLogger.GetSink()).To(BeIdenticalTo(newSink))
		})
	}
}

func Test_extendLoggerWithPipelineRunInfo_PipelineRunIsNil(t *testing.T) {
	// SETUP
	g := NewGomegaWithT(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSink := logrtesting.NewMockLogSink(mockCtrl)
	mockSink.EXPECT().Init(gomock.Any())
	// no other calls expected

	logger := logr.New(mockSink)

	// EXERCISE + VERIFY
	g.Expect(func() {
		extendLoggerWithPipelineRunInfo(logger, nil, nil)
	}).To(
		Panic(),
	)
}

func Test_ExtendContextLoggerWithPipelineRunInfo(t *testing.T) {
	// SETUP
	g := NewGomegaWithT(t)

	pipelineRun := &api.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-2",
			Namespace: "run-namespace-2",
			UID:       k8sapitypes.UID("uid-2"),
		},
		Status: api.PipelineStatus{
			State:              api.StatePreparing,
			Namespace:          "exec-namespace-2",
			AuxiliaryNamespace: "exec-aux-namespace-2",
		},
	}

	customLoggingDetails := func(pipelineRunPassed *api.PipelineRun) []any {
		g.Expect(pipelineRunPassed).To(BeIdenticalTo(pipelineRun))
		return []any{
			"custom2", "custom2Value",
			"custom1", "custom1Value",
		}
	}

	expectedWithKVs := []interface{}{
		"pipelineRun", klog.ObjectRef{Name: "run-2", Namespace: "run-namespace-2"},
		"pipelineRunUID", k8sapitypes.UID("uid-2"),
		"pipelineRunState", api.StatePreparing,
		"pipelineRunExecNamespace", "exec-namespace-2",
		"pipelineRunExecAuxNamespace", "exec-aux-namespace-2",
		"custom2", "custom2Value",
		"custom1", "custom1Value",
	}

	mockCtrl := gomock.NewController(t)

	origSink := logrtesting.NewMockLogSink(mockCtrl)
	newSink := logrtesting.NewMockLogSink(mockCtrl)

	gomock.InOrder(
		origSink.EXPECT().Init(gomock.Any()),
		origSink.EXPECT().WithValues(gomock.Any()).DoAndReturn(
			func(args ...interface{}) logr.LogSink {
				g.Expect(args).To(HaveExactElements(expectedWithKVs...))
				return newSink
			},
		),
	)

	logger := logr.New(origSink)

	type baseCtxKey struct{}
	baseCtxValue := 94586724935743
	baseCtx := context.WithValue(context.Background(), baseCtxKey{}, baseCtxValue)
	ctx := logr.NewContext(baseCtx, logger)

	config := &cfg.PipelineRunsConfigStruct{
		CustomLoggingDetails: customLoggingDetails,
	}
	ctx = cfg.NewContextWithConfig(ctx, config)

	// EXERCISE
	resultCtx, resultLogger := ExtendContextLoggerWithPipelineRunInfo(
		ctx, pipelineRun,
	)

	// VERIFY
	g.Expect(resultCtx).NotTo(BeIdenticalTo(ctx))
	g.Expect(logr.FromContext(resultCtx)).To(BeIdenticalTo(resultLogger))
	g.Expect(resultCtx.Value(baseCtxKey{})).To(BeIdenticalTo(baseCtxValue))

	g.Expect(resultLogger).NotTo(BeIdenticalTo(logger))
	g.Expect(resultLogger.GetSink()).To(BeIdenticalTo(newSink))
}

func Test_ExtendContextLoggerWithPipelineRunInfo_ContextIsNil(t *testing.T) {
	// SETUP
	g := NewGomegaWithT(t)

	pipelineRun := &api.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-2",
			Namespace: "run-namespace-2",
			UID:       k8sapitypes.UID("uid-2"),
		},
		Status: api.PipelineStatus{
			State:              api.StatePreparing,
			Namespace:          "exec-namespace-2",
			AuxiliaryNamespace: "exec-aux-namespace-2",
		},
	}

	nilCtx := (context.Context)(nil)

	// EXERCISE + VERIFY
	g.Expect(func() {
		ExtendContextLoggerWithPipelineRunInfo(nilCtx, pipelineRun)
	}).To(
		Panic(),
	)
}

func Test_ExtendContextLoggerWithPipelineRunInfo_ContextHasNoLogger(t *testing.T) {
	// SETUP
	g := NewGomegaWithT(t)

	pipelineRun := &api.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-2",
			Namespace: "run-namespace-2",
			UID:       k8sapitypes.UID("uid-2"),
		},
		Status: api.PipelineStatus{
			State:              api.StatePreparing,
			Namespace:          "exec-namespace-2",
			AuxiliaryNamespace: "exec-aux-namespace-2",
		},
	}

	ctxWithoutLogger := context.Background()

	// EXERCISE + VERIFY
	g.Expect(func() {
		ExtendContextLoggerWithPipelineRunInfo(ctxWithoutLogger, pipelineRun)
	}).To(
		Panic(),
	)
}

func Test_ExtendContextLoggerWithPipelineRunInfo_ContextHasNoConfig(t *testing.T) {
	// SETUP
	g := NewGomegaWithT(t)

	pipelineRun := &api.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-2",
			Namespace: "run-namespace-2",
			UID:       k8sapitypes.UID("uid-2"),
		},
		Status: api.PipelineStatus{
			State:              api.StatePreparing,
			Namespace:          "exec-namespace-2",
			AuxiliaryNamespace: "exec-aux-namespace-2",
		},
	}

	expectedWithKVs := []interface{}{
		"pipelineRun", klog.ObjectRef{Name: "run-2", Namespace: "run-namespace-2"},
		"pipelineRunUID", k8sapitypes.UID("uid-2"),
		"pipelineRunState", api.StatePreparing,
		"pipelineRunExecNamespace", "exec-namespace-2",
		"pipelineRunExecAuxNamespace", "exec-aux-namespace-2",
	}

	mockCtrl := gomock.NewController(t)

	origSink := logrtesting.NewMockLogSink(mockCtrl)
	newSink := logrtesting.NewMockLogSink(mockCtrl)

	gomock.InOrder(
		origSink.EXPECT().Init(gomock.Any()),
		origSink.EXPECT().WithValues(gomock.Any()).DoAndReturn(
			func(args ...interface{}) logr.LogSink {
				g.Expect(args).To(HaveExactElements(expectedWithKVs...))
				return newSink
			},
		),
	)

	logger := logr.New(origSink)

	type baseCtxKey struct{}
	baseCtxValue := 94586724935743
	baseCtx := context.WithValue(context.Background(), baseCtxKey{}, baseCtxValue)
	ctx := logr.NewContext(baseCtx, logger)
	// ctx does NOT carry a pipeline runs config

	// EXERCISE
	resultCtx, resultLogger := ExtendContextLoggerWithPipelineRunInfo(
		ctx, pipelineRun,
	)

	// VERIFY
	g.Expect(resultCtx).NotTo(BeIdenticalTo(ctx))
	g.Expect(logr.FromContext(resultCtx)).To(BeIdenticalTo(resultLogger))
	g.Expect(resultCtx.Value(baseCtxKey{})).To(BeIdenticalTo(baseCtxValue))

	g.Expect(resultLogger).NotTo(BeIdenticalTo(logger))
	g.Expect(resultLogger.GetSink()).To(BeIdenticalTo(newSink))
}

func Test_ExtendContextLoggerWithPipelineRunInfo_ErrorGettingConfigFromContext(t *testing.T) {
	// SETUP
	g := NewGomegaWithT(t)

	pipelineRun := &api.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-2",
			Namespace: "run-namespace-2",
			UID:       k8sapitypes.UID("uid-2"),
		},
		Status: api.PipelineStatus{
			State:              api.StatePreparing,
			Namespace:          "exec-namespace-2",
			AuxiliaryNamespace: "exec-aux-namespace-2",
		},
	}

	expectedWithKVs := []interface{}{
		"pipelineRun", klog.ObjectRef{Name: "run-2", Namespace: "run-namespace-2"},
		"pipelineRunUID", k8sapitypes.UID("uid-2"),
		"pipelineRunState", api.StatePreparing,
		"pipelineRunExecNamespace", "exec-namespace-2",
		"pipelineRunExecAuxNamespace", "exec-aux-namespace-2",
	}

	mockCtrl := gomock.NewController(t)

	origSink := logrtesting.NewMockLogSink(mockCtrl)
	newSink := logrtesting.NewMockLogSink(mockCtrl)

	gomock.InOrder(
		origSink.EXPECT().Init(gomock.Any()),
		origSink.EXPECT().WithValues(gomock.Any()).DoAndReturn(
			func(args ...interface{}) logr.LogSink {
				g.Expect(args).To(HaveExactElements(expectedWithKVs...))
				return newSink
			},
		),
	)

	logger := logr.New(origSink)

	type baseCtxKey struct{}
	baseCtxValue := 94586724935743
	baseCtx := context.WithValue(context.Background(), baseCtxKey{}, baseCtxValue)
	ctx := logr.NewContext(baseCtx, logger)

	mockClientFactory := fake.NewClientFactory( /* no objects -> error loading config */ )
	ctx = cfg.NewContext(ctx, mockClientFactory)

	// EXERCISE
	resultCtx, resultLogger := ExtendContextLoggerWithPipelineRunInfo(
		ctx, pipelineRun,
	)

	// VERIFY
	g.Expect(resultCtx).NotTo(BeIdenticalTo(ctx))
	g.Expect(logr.FromContext(resultCtx)).To(BeIdenticalTo(resultLogger))
	g.Expect(resultCtx.Value(baseCtxKey{})).To(BeIdenticalTo(baseCtxValue))

	g.Expect(resultLogger).NotTo(BeIdenticalTo(logger))
	g.Expect(resultLogger.GetSink()).To(BeIdenticalTo(newSink))
}

func Test_ExtendContextLoggerWithPipelineRunInfo_PipelineRunIsNil(t *testing.T) {
	// SETUP
	g := NewGomegaWithT(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSink := logrtesting.NewMockLogSink(mockCtrl)
	mockSink.EXPECT().Init(gomock.Any())
	// no other calls expected

	logger := logr.New(mockSink)
	ctx := logr.NewContext(context.Background(), logger)

	// EXERCISE + VERIFY
	g.Expect(func() {
		ExtendContextLoggerWithPipelineRunInfo(ctx, nil)
	}).To(
		Panic(),
	)
}
