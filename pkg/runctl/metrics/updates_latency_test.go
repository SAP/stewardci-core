package metrics

import (
	"testing"

	"gotest.tools/v3/assert"
)

func Test_UpdatesLatency_isInitialized(t *testing.T) {
	t.Parallel()

	// VERIFY
	assert.Assert(t, *(UpdatesLatency.(*updatesLatency)) != updatesLatency{})
}

// TODO add tests for updatesLatency
