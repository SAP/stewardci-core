package builder

import (
	"gotest.tools/assert"
	"testing"
)

func Test_PipelineRun_Builder(t *testing.T) {
	pipelineRun := PipelineRun("namespace1",
		PipelineRunSpec(
			JenkinsFileSpec("https://foo.bar", "revision1", "path1")))
	assert.Equal(t, "https://foo.bar", pipelineRun.Spec.JenkinsFile.URL)
	assert.Equal(t, "revision1", pipelineRun.Spec.JenkinsFile.Revision)
	assert.Equal(t, "path1", pipelineRun.Spec.JenkinsFile.Path)
}
