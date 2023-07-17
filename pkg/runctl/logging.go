package runctl

import (
	"context"

	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/go-logr/logr"
	klog "k8s.io/klog/v2"
)

func extendContextLoggerWithPipelineRunInfo(ctx context.Context, pipelineRun k8s.PipelineRun) (context.Context, logr.Logger) {

	if ctx == nil {
		ctx = context.Background()
	}

	logger := extendLoggerWithPipelineRunInfo(klog.FromContext(ctx), pipelineRun)

	return klog.NewContext(ctx, logger), logger
}

func extendLoggerWithPipelineRunInfo(logger logr.Logger, pipelineRun k8s.PipelineRun) logr.Logger {

	kvs := getPipelineRunInfoForLogging(pipelineRun)

	return logger.WithValues(kvs...)
}

func getPipelineRunInfoForLogging(run k8s.PipelineRun) []interface{} {
	if run == nil {
		return nil
	}

	runStatus := run.GetStatus()

	kvs := []interface{}{
		"pipelineRun", klog.KObj(run),
		"pipelineRunUID", run.GetAPIObject().ObjectMeta.UID,
		"pipelineRunState", runStatus.State,
		"pipelineRunExecutionNamespace", runStatus.Namespace,
	}

	if runStatus.AuxiliaryNamespace != "" {
		kvs = append(kvs,
			"pipelineRunExecutionAuxiliaryNamespace", runStatus.AuxiliaryNamespace,
		)
	}
	return kvs
}
