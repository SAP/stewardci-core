package k8srestclient

import k8sclientmetrics "k8s.io/client-go/tools/metrics"

func init() {
	// TODO map more metrics
	k8sclientmetrics.Register(
		k8sclientmetrics.RegisterOpts{
			RateLimiterLatency: rateLimitLatencyInstance,
			RequestLatency:     requestLatencyInstance,
			RequestResult:      requestResultsInstance,
		},
	)
}
