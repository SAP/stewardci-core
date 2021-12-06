package metrics

import (
	"testing"

	"gotest.tools/assert"
)

func Test_ControllerHeartbeats_isInitialized(t *testing.T) {
	t.Parallel()

	// VERIFY
	assert.Assert(t, *(ControllerHeartbeats.(*controllerHeartbeats)) != controllerHeartbeats{})
}

// TODO add tests for controllerHeartbeats
