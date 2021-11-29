package metrics

import (
	metricswq "github.com/SAP/stewardci-core/pkg/metrics/workqueue"
)

func init() {
	metricswq.RegisterNameProvider(
		metricswq.NameProviderFunc(
			func(queueName string) (string, bool) {
				if queueName == WorkqueueName {
					return subsystemForWorkqueue, true
				}
				return "", false
			},
		),
	)
}
