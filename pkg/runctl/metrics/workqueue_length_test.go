package metrics

import (
	"testing"

	"gotest.tools/assert"
)

func Test_WorkqueueLength_isInitialized(t *testing.T) {
	t.Parallel()

	// VERIFY
	assert.Assert(t, *(WorkqueueLength.(*workqueueLength)) != workqueueLength{})
}

// TODO add tests for workqueueLength
