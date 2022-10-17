package k8srestclient

import (
	"testing"

	"gotest.tools/v3/assert"
)

func Test_requestResultsInstance_isInitialized(t *testing.T) {
	t.Parallel()

	// VERIFY
	assert.Assert(t, *requestResultsInstance != requestResults{})
}

// TODO add tests for requestResults
