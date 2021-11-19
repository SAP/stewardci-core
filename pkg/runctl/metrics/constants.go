package metrics

import "github.com/SAP/stewardci-core/pkg/metrics"

const (
	subsystem             = metrics.Subsystem + "_pipelineruns"
	subsystemForWorkqueue = subsystem + "_workqueue"

	// WorkqueueName is the name of the run controller workqueue.
	// It is required by the metrics adapter for workqueues.
	WorkqueueName = "runctl"
)
