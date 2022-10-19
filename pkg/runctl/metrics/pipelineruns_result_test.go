package metrics

import (
	"testing"

	"gotest.tools/v3/assert"
)

func Test_PipelineRunsResult_isInitialized(t *testing.T) {
	t.Parallel()

	// VERIFY
	assert.Assert(t, *(PipelineRunsResult.(*pipelineRunsResult)) != pipelineRunsResult{})
}

// TODO add tests for pipelineRunsResult
