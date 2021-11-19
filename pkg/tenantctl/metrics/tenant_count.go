package metrics

import (
	"sync"

	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// TenantCount refects the current number of tenants
	// in the system.
	TenantCount SettableGaugeMetric = &tenantCount{}
)

func init() {
	TenantCount.(*tenantCount).init()
}

type tenantCount struct {
	initOnlyOnce sync.Once
	metric       prometheus.Gauge
}

func (m *tenantCount) init() {
	m.initOnlyOnce.Do(func() {
		m.metric = prometheus.NewGauge(prometheus.GaugeOpts{
			// TODO use metric name prefixes consistently
			//Subsystem: subsystem,
			//Name:      "count",
			Name: "steward_tenants_total",
			Help: "The current number of tenants in the system.",
		})
		metrics.Registerer().MustRegister(m.metric)
	})
}

func (m *tenantCount) Set(value float64) {
	m.metric.Set(value)
}
