package builder

import (
	"gotest.tools/assert"
	"testing"
)

func Test_PipelineRunBuilder_Jenkinsfile(t *testing.T) {
	pipelineRun := PipelineRun("prefix1", "namespace1",
		PipelineRunSpec(
			JenkinsFileSpec("https://foo.bar", "revision1", "path1")))
	assert.Equal(t, "https://foo.bar", pipelineRun.Spec.JenkinsFile.URL)
	assert.Equal(t, "revision1", pipelineRun.Spec.JenkinsFile.Revision)
	assert.Equal(t, "path1", pipelineRun.Spec.JenkinsFile.Path)
}

func Test_PipelineRunBuilder_ArgSpec(t *testing.T) {
	pipelineRun := PipelineRun("prefix1", "namespace1",
		PipelineRunSpec(
			ArgSpec("foo", "bar"),
			ArgSpec("baz", "bum")))
	assert.Equal(t, "bar", pipelineRun.Spec.Args["foo"])
	assert.Equal(t, "bum", pipelineRun.Spec.Args["baz"])
}
