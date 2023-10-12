package runctl

import (
	"context"

	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/go-logr/logr"
	klog "k8s.io/klog/v2"
)

// extendContextLoggerWithPipelineRunInfo calls extendLoggerWithPipelineRunInfo
// with the logger from the given context.
// Panics if the given context has no logger attached or the given pipeline run
// is nil.
// Returns both a new context with the enhanced logger and the enhanced logger
// so that callers can directly use what they need.
func extendContextLoggerWithPipelineRunInfo(ctx context.Context, pipelineRun *v1alpha1.PipelineRun, labelsToLog map[string]string) (context.Context, logr.Logger) {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		panic(err)
	}
	logger = extendLoggerWithPipelineRunInfo(logger, pipelineRun, labelsToLog)
	return klog.NewContext(ctx, logger), logger
}

// extendLoggerWithPipelineRunInfo attaches some data of the given pipelineRun
// as values to the given logger. The enhanced logger is returned.
// Panics if the given pipeline run is nil.
func extendLoggerWithPipelineRunInfo(logger logr.Logger, pipelineRun *v1alpha1.PipelineRun, labelsToLog map[string]string) logr.Logger {
	kvs := getPipelineRunInfoForLogging(pipelineRun, labelsToLog)
	return logger.WithValues(kvs...)
}

func getPipelineRunInfoForLogging(run *v1alpha1.PipelineRun, labelsToLog map[string]string) []interface{} {
	kvs := []interface{}{
		"pipelineRun", klog.KObj(&run.ObjectMeta),
		"pipelineRunUID", run.ObjectMeta.UID,
		"pipelineRunState", run.Status.State,
	}
	if run.Status.Namespace != "" {
		kvs = append(kvs,
			"pipelineRunExecNamespace", run.Status.Namespace,
		)
	}
	if run.Status.AuxiliaryNamespace != "" {
		kvs = append(kvs,
			"pipelineRunExecAuxNamespace", run.Status.AuxiliaryNamespace,
		)
	}
	labels := run.GetLabels()
	if labels != nil {
		for labelToLog, shortLabelKey := range labelsToLog {

			value := labels[labelToLog]
			if value != "" {
				kvs = append(kvs, shortLabelKey, value)
			}
		}
	}
	return kvs
}
