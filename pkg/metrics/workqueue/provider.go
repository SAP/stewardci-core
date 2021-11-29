package workqueue

import (
	"strings"

	"github.com/SAP/stewardci-core/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/util/workqueue"
)

func init() {
	workqueue.SetProvider(&prometheusMetricsProvider{})
}

type prometheusMetricsProvider struct {
	cache cache
}

func (p *prometheusMetricsProvider) NewDepthMetric(queueName string) workqueue.GaugeMetric {
	metricName := p.metricName(queueName, "depth")
	return p.cache.GetOrCreate(metricName, func() interface{} {
		metric := prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: metricName,
				Help: "The current depth of the workqueue.",
			},
		)
		metrics.Registerer().MustRegister(metric)
		return metric
	}).(workqueue.GaugeMetric)
}

func (p *prometheusMetricsProvider) NewAddsMetric(queueName string) workqueue.CounterMetric {
	metricName := p.metricName(queueName, "adds_total")
	return p.cache.GetOrCreate(metricName, func() interface{} {
		metric := prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: metricName,
				Help: "The number of entries added to the workqueue over time.",
			},
		)
		metrics.Registerer().MustRegister(metric)
		return metric
	}).(workqueue.CounterMetric)
}

func (p *prometheusMetricsProvider) NewLatencyMetric(queueName string) workqueue.HistogramMetric {

	buckets := func() []float64 {
		list := make([]float64, 0, 14)
		for i := 1e-3; i <= 1e+3; i *= 10.0 {
			list = append(list, i, i*5.0)
		}
		return list
	}

	metricName := p.metricName(queueName, "latency_seconds")
	return p.cache.GetOrCreate(metricName, func() interface{} {
		metric := prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: metricName,
			Help: "A histogram of queuing latency." +
				" The latency is the time an item was waiting in the queue until processing the item started." +
				" The processing time is therefore not included.",
			Buckets: buckets(),
		})
		metrics.Registerer().MustRegister(metric)
		return metric
	}).(workqueue.HistogramMetric)
}

func (p *prometheusMetricsProvider) NewWorkDurationMetric(queueName string) workqueue.HistogramMetric {

	buckets := func() []float64 {
		list := make([]float64, 0, 14)
		for i := 1e-3; i <= 1e+3; i *= 10.0 {
			list = append(list, i, i*5.0)
		}
		return list
	}

	metricName := p.metricName(queueName, "workduration_seconds")
	return p.cache.GetOrCreate(metricName, func() interface{} {
		metric := prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: metricName,
			Help: "A histogram of per-item processing times." +
				" The processing time of a queue item is the time the application worked on it, but not the time it has been waiting in the queue.",
			Buckets: buckets(),
		})
		metrics.Registerer().MustRegister(metric)
		return metric
	}).(workqueue.HistogramMetric)
}

func (p *prometheusMetricsProvider) NewUnfinishedWorkSecondsMetric(queueName string) workqueue.SettableGaugeMetric {
	metricName := p.metricName(queueName, "unfinished_workduration_seconds")
	return p.cache.GetOrCreate(metricName, func() interface{} {
		metric := prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: metricName,
				Help: "The sum of processing time spent on items still in the queue." +
					" Once an item gets removed from the workqueue, it does not count into this metric anymore.",
			},
		)
		metrics.Registerer().MustRegister(metric)
		return metric
	}).(workqueue.SettableGaugeMetric)
}

func (p *prometheusMetricsProvider) NewLongestRunningProcessorSecondsMetric(queueName string) workqueue.SettableGaugeMetric {
	metricName := p.metricName(queueName, "longest_running_processor_seconds")
	return p.cache.GetOrCreate(metricName, func() interface{} {
		metric := prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: metricName,
				Help: "The longest processing time spent on a single item that is still in the queue.",
			},
		)
		metrics.Registerer().MustRegister(metric)
		return metric
	}).(workqueue.SettableGaugeMetric)
}

func (p *prometheusMetricsProvider) NewRetriesMetric(queueName string) workqueue.CounterMetric {
	metricName := p.metricName(queueName, "retry_count_total")
	return p.cache.GetOrCreate(metricName, func() interface{} {
		metric := prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: metricName,
				Help: "The total number of retries needed to process queue items.",
			},
		)
		metrics.Registerer().MustRegister(metric)
		return metric
	}).(workqueue.CounterMetric)
}

func (p *prometheusMetricsProvider) metricName(queueName, simpleName string) string {
	nameParts := []string{
		nameProvidersInstance.MustGetSubsystemFor(queueName),
		simpleName,
	}
	return strings.Join(nameParts, "_")
}
