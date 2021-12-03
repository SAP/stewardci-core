package metrics

import (
	"testing"

	"gotest.tools/assert"
)

func Test_PipelineRunsStarted_isInitialized(t *testing.T) {
	t.Parallel()

	// VERIFY
	assert.Assert(t, *(PipelineRunsStarted.(*pipelineRunsStarted)) != pipelineRunsStarted{})
}

// TODO add tests for pipelineRunsStarted
