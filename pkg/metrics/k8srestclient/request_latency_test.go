package k8srestclient

import (
	"testing"

	"gotest.tools/assert"
)

func Test_requestLatencyInstance_isInitialized(t *testing.T) {
	t.Parallel()

	// VERIFY
	assert.Assert(t, *requestLatencyInstance != requestLatency{})
}

// TODO add tests for requestLatency
