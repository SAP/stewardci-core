package k8srestclient

import k8sclientmetrics "k8s.io/client-go/tools/metrics"

func init() {
	// TODO upgrade to newer version of client-go and map more metrics
	k8sclientmetrics.Register(
		requestLatencyInstance,
		requestResultsInstance,
	)
}
