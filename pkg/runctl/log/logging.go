package log

import (
	"context"

	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/runctl/cfg"
	"github.com/SAP/stewardci-core/pkg/runctl/log/custom"
	"github.com/go-logr/logr"
	klog "k8s.io/klog/v2"
)

// ExtendContextLoggerWithPipelineRunInfo calls extendLoggerWithPipelineRunInfo
// with the logger from the given context.
// Panics if the given context has no logger attached or the given pipeline run
// is nil.
// Returns both a new context with the enhanced logger and the enhanced logger
// so that callers can directly use what they need.
func ExtendContextLoggerWithPipelineRunInfo(ctx context.Context, pipelineRun *v1alpha1.PipelineRun) (context.Context, logr.Logger) {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		panic(err)
	}
	var customLoggingDetails custom.LoggingDetailsProvider
	config, err := cfg.FromContext(ctx)
	if err == nil && config != nil {
		customLoggingDetails = config.CustomLoggingDetails
	}

	logger = extendLoggerWithPipelineRunInfo(logger, pipelineRun, customLoggingDetails)
	return klog.NewContext(ctx, logger), logger
}

// extendLoggerWithPipelineRunInfo attaches some data of the given pipelineRun
// as values to the given logger. The enhanced logger is returned.
// Panics if the given pipeline run is nil.
func extendLoggerWithPipelineRunInfo(logger logr.Logger, pipelineRun *v1alpha1.PipelineRun, customLoggingDetails custom.LoggingDetailsProvider) logr.Logger {
	kvs := getPipelineRunInfoForLogging(pipelineRun, customLoggingDetails)
	return logger.WithValues(kvs...)
}

func getPipelineRunInfoForLogging(run *v1alpha1.PipelineRun, customLoggingDetails custom.LoggingDetailsProvider) []interface{} {
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
	if customLoggingDetails != nil {
		kvs = append(kvs, customLoggingDetails(run)...)
	}
	return kvs
}
