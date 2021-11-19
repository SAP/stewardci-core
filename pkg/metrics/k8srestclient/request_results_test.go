package k8srestclient

import (
	"testing"

	"gotest.tools/assert"
)

func Test_requestResultsInstance_isInitialized(t *testing.T) {
	t.Parallel()

	// VERIFY
	assert.Assert(t, *requestResultsInstance != requestResults{})
}

// TODO add tests for requestResults
