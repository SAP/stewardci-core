package metrics

import (
	"fmt"
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
	// TODO remove when deprecated long enough
	metricOld prometheus.Gauge
}

func (m *tenantCount) init() {
	m.initOnlyOnce.Do(func() {
		m.metric = prometheus.NewGauge(prometheus.GaugeOpts{
			Subsystem: subsystem,
			Name:      "count_total",
			Help:      "The current number of tenants in the system.",
		})
		metrics.Registerer().MustRegister(m.metric)

		m.metricOld = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "steward_tenants_total",
			Help: fmt.Sprintf("Deprecated! Use '%s_count' instead.", subsystem),
		})
		metrics.Registerer().MustRegister(m.metricOld)
	})
}

func (m *tenantCount) Set(value float64) {
	m.metric.Set(value)
	m.metricOld.Set(value)
}
