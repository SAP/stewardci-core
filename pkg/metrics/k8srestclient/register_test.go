package k8srestclient

import (
	"testing"

	"gotest.tools/assert"
	k8sclientmetrics "k8s.io/client-go/tools/metrics"
)

func TestRegistration(t *testing.T) {
	t.Parallel()

	// VERIFY
	assert.Equal(t, k8sclientmetrics.RequestLatency, requestLatencyInstance)
	assert.Equal(t, k8sclientmetrics.RequestResult, requestResultsInstance)
}
