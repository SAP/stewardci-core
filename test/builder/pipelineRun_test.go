package builder

import (
	"gotest.tools/assert"
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
)

func Test_PipelineRunBuilder_Jenkinsfile(t *testing.T) {
	pipelineRun := PipelineRun("prefix1", "namespace1",
		PipelineRunSpec(
			JenkinsFileSpec("https://foo.bar", "path1")))
	assert.Equal(t, "https://foo.bar", pipelineRun.Spec.JenkinsFile.URL)
	assert.Equal(t, "master", pipelineRun.Spec.JenkinsFile.Revision)
	assert.Equal(t, "path1", pipelineRun.Spec.JenkinsFile.Path)
}

func Test_PipelineRunBuilder_JenkinsfileWithOps(t *testing.T) {
	pipelineRun := PipelineRun("prefix1", "namespace1",
		PipelineRunSpec(
			JenkinsFileSpec("https://foo.bar", "path1",
				Revision("revision1"),
				RepoAuthSecret("secret1"),
			),
		),
	)
	assert.DeepEqual(t, api.JenkinsFile{URL: "https://foo.bar",
		Revision:       "revision1",
		Path:           "path1",
		RepoAuthSecret: "secret1",
	}, pipelineRun.Spec.JenkinsFile)
}

func Test_PipelineRunBuilder_ArgSpec(t *testing.T) {
	pipelineRun := PipelineRun("prefix1", "namespace1",
		PipelineRunSpec(
			ArgSpec("foo", "bar"),
			ArgSpec("baz", "bum"),
		),
	)
	assert.DeepEqual(t, map[string]string{"foo": "bar", "baz": "bum"}, pipelineRun.Spec.Args)
}

func Test_PipelineRunBuilder_Secret(t *testing.T) {
	pipelineRun := PipelineRun("prefix1", "namespace1",
		PipelineRunSpec(
			Secret("foo"),
			ImagePullSecret("pull1"),
			Secret("bar"),
			ImagePullSecret("pull2"),
		),
	)
	assert.DeepEqual(t, []string{"foo", "bar"}, pipelineRun.Spec.Secrets)
	assert.DeepEqual(t, []string{"pull1", "pull2"}, pipelineRun.Spec.ImagePullSecrets)
}

func Test_PipelineRunBuilder_RunDetails(t *testing.T) {
	pipelineRun := PipelineRun("prefix1", "namespace1",
		PipelineRunSpec(
			RunDetails("jobName1", "cause1", 42),
		),
	)
	assert.DeepEqual(t, &api.PipelineRunDetails{
		JobName:        "jobName1",
		SequenceNumber: 42,
		Cause:          "cause1",
	}, pipelineRun.Spec.RunDetails)
}

func Test_PipelineRunAbort(t *testing.T) {
	pipelineRun := PipelineRun("prefix1", "namespace1",
		PipelineRunSpec(
			Abort(),
		),
	)
	assert.Equal(t, api.IntentAbort,
		pipelineRun.Spec.Intent)
}
