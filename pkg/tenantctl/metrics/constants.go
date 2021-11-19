package metrics

import "github.com/SAP/stewardci-core/pkg/metrics"

const (
	subsystem             = metrics.Subsystem + "_tenants"
	subsystemForWorkqueue = subsystem + "_workqueue"

	// WorkqueueName is the name of the tenant controller workqueue.
	// It is required by the metrics adapter for workqueues.
	WorkqueueName = "tenantctl"
)
