package k8srestclient

import (
	"testing"

	"gotest.tools/assert"
)

func Test_rateLimitLatencyInstance_isInitialized(t *testing.T) {
	t.Parallel()

	// VERIFY
	assert.Assert(t, *rateLimitLatencyInstance != rateLimitLatency{})
}

// TODO add tests for rateLimitLatency
