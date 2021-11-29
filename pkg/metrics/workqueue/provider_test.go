package workqueue

import (
	"k8s.io/client-go/util/workqueue"
)

var _ workqueue.MetricsProvider = (*prometheusMetricsProvider)(nil)
