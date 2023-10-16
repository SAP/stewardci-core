package runctl

import (
	"context"
	"testing"

	logrtesting "github.com/SAP/stewardci-core/internal/logr/testing"
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/runctl/cfg"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	assert "gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sapitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
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
				emptyLoggingDetais,
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
		extendLoggerWithPipelineRunInfo(logger, nil, emptyLoggingDetais)
	}).To(
		Panic(),
	)
}

func Test_extendContextLoggerWithPipelineRunInfo(t *testing.T) {
	// SETUP
	const (
		annotationKey = "annotationKey"
		labelKey      = "labelKey"
		logKey1       = "key1"
		logKey2       = "key2"
	)
	pipelineRun := &api.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "run-2",
			Namespace:   "run-namespace-2",
			UID:         k8sapitypes.UID("uid-2"),
			Annotations: map[string]string{annotationKey: "annotationValue1"},
			Labels:      map[string]string{labelKey: "labelValue1"},
		},
		Status: api.PipelineStatus{
			State:              api.StatePreparing,
			Namespace:          "exec-namespace-2",
			AuxiliaryNamespace: "exec-aux-namespace-2",
		},
	}

	loggingDetails := map[string]cfg.PipelineRunAccessor{
		logKey1: cfg.PipelineRunAccessor{
			Kind: cfg.KindAnnotationAccessor,
			Name: annotationKey,
		},
		logKey2: cfg.PipelineRunAccessor{
			Kind: cfg.KindLabelAccessor,
			Name: labelKey,
		},
	}

	expectedWithKVs := []interface{}{
		"pipelineRun", klog.ObjectRef{Name: "run-2", Namespace: "run-namespace-2"},
		"pipelineRunUID", k8sapitypes.UID("uid-2"),
		"pipelineRunState", api.StatePreparing,
		"pipelineRunExecNamespace", "exec-namespace-2",
		"pipelineRunExecAuxNamespace", "exec-aux-namespace-2",
		logKey1, "annotationValue1",
		logKey2, "labelValue1",
	}

	g := NewGomegaWithT(t)
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

	// EXERCISE
	resultCtx, resultLogger := extendContextLoggerWithPipelineRunInfo(
		ctx, pipelineRun, loggingDetails,
	)

	// VERIFY
	g.Expect(resultCtx).NotTo(BeIdenticalTo(ctx))
	g.Expect(logr.FromContext(resultCtx)).To(BeIdenticalTo(resultLogger))
	g.Expect(resultCtx.Value(baseCtxKey{})).To(BeIdenticalTo(baseCtxValue))

	g.Expect(resultLogger).NotTo(BeIdenticalTo(logger))
	g.Expect(resultLogger.GetSink()).To(BeIdenticalTo(newSink))
}

func Test_extendContextLoggerWithPipelineRunInfo_ContextIsNil(t *testing.T) {
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
		extendContextLoggerWithPipelineRunInfo(nilCtx, pipelineRun, emptyLoggingDetais)
	}).To(
		Panic(),
	)
}

func Test_extendContextLoggerWithPipelineRunInfo_ContextHasNoLogger(t *testing.T) {
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
		extendContextLoggerWithPipelineRunInfo(ctxWithoutLogger, pipelineRun, emptyLoggingDetais)
	}).To(
		Panic(),
	)
}

func Test_extendContextLoggerWithPipelineRunInfo_PipelineRunIsNil(t *testing.T) {
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
		extendContextLoggerWithPipelineRunInfo(ctx, nil, emptyLoggingDetais)
	}).To(
		Panic(),
	)
}

func Test_getValueByAccessor(t *testing.T) {
	const (
		annotationKey1       = "ak1"
		annotationKey2       = "ak2"
		annotationKeyUnknown = "ak3"
		annotationValue1     = "av1"
		annotationValue2     = "av2"
		labelKey1            = "lk1"
		labelKey2            = "lk2"
		labelKeyUnknown      = "lk3"
		labelValue1          = "lv1"
		labelValue2          = "lv2"
	)
	run :=
		&api.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					annotationKey1: annotationValue1,
					annotationKey2: annotationValue2,
				},
				Labels: map[string]string{
					labelKey1: labelValue1,
					labelKey2: labelValue2,
				},
			},
		}

	for _, test := range []struct {
		name           string
		accessor       cfg.PipelineRunAccessor
		expectedResult string
	}{
		{
			name:           "empty",
			expectedResult: "",
		},
		{
			name: "annotation 1",
			accessor: cfg.PipelineRunAccessor{
				Kind: cfg.KindAnnotationAccessor,
				Name: annotationKey1,
			},
			expectedResult: annotationValue1,
		},
		{
			name: "annotation 2",
			accessor: cfg.PipelineRunAccessor{
				Kind: cfg.KindAnnotationAccessor,
				Name: annotationKey2,
			},
			expectedResult: annotationValue2,
		},
		{
			name: "annotation empyt",
			accessor: cfg.PipelineRunAccessor{
				Kind: cfg.KindAnnotationAccessor,
				Name: "",
			},
		},
		{
			name: "annotation unknown",
			accessor: cfg.PipelineRunAccessor{
				Kind: cfg.KindAnnotationAccessor,
				Name: annotationKeyUnknown,
			},
		},
		{
			name: "label 1",
			accessor: cfg.PipelineRunAccessor{
				Kind: cfg.KindLabelAccessor,
				Name: labelKey1,
			},
			expectedResult: labelValue1,
		},
		{
			name: "label 2",
			accessor: cfg.PipelineRunAccessor{
				Kind: cfg.KindLabelAccessor,
				Name: labelKey2,
			},
			expectedResult: labelValue2,
		},
		{
			name: "label empty",
			accessor: cfg.PipelineRunAccessor{
				Kind: cfg.KindLabelAccessor,
				Name: "",
			},
		},
		{
			name: "label unknown",
			accessor: cfg.PipelineRunAccessor{
				Kind: cfg.KindLabelAccessor,
				Name: labelKeyUnknown,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test := test
			t.Parallel()

			// EXERCISE
			result := getValueByAccessor(run, test.accessor)

			// VERIFY
			assert.Equal(t, test.expectedResult, result)
		})
	}
}
