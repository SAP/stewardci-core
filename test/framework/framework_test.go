//go:build frameworktest
// +build frameworktest

package framework

import (
	"testing"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/test/builder"
	"github.com/SAP/stewardci-core/test/shared"
	v1 "k8s.io/api/core/v1"
)

// frameworkTestBuilders is a list of all test builders
var frameworkTestBuilders = []PipelineRunTestBuilder{
	PipelineRunSleepTooLong,
	PipelineRunWrongExpect,
	PipelineRunWrongName,
	PipelineRunWithSecretNameConflict,
}

func Test_FirstFinishBeforeSecondStarts(t *testing.T) {
	test := TestPlan{TestBuilder: PipelineRunWrongName,
		Count:         2,
		CreationDelay: time.Second * 1,
		Name:          "FirstFinishBeforeSecondStarts",
	}
	ctx := Setup(t)
	executePipelineRunTests(ctx, t, test)
}

func Test_FrameworkTest(t *testing.T) {
	t.Parallel()
	allTests := make([]TestPlan, len(frameworkTestBuilders))
	for i, pipelinerunTestBuilder := range frameworkTestBuilders {
		allTests[i] = TestPlan{TestBuilder: pipelinerunTestBuilder,
			Count: 1,
		}
	}
	ctx := Setup(t)
	executePipelineRunTests(ctx, t, allTests...)
}

// PipelineRunSleepTooLong is a PipelineRunTestBuilder to test if Timeout works correctly
func PipelineRunSleepTooLong(Namespace string, runID *api.CustomJSON) PipelineRunTest {
	return PipelineRunTest{
		PipelineRun: builder.PipelineRun("sleeptoolong-", Namespace,
			builder.PipelineRunSpec(
				builder.LoggingWithRunID(runID),
				builder.JenkinsFileSpec(shared.ExamplePipelineRepoURL,
					"sleep/Jenkinsfile", shared.ExamplePipelineRepoRevision),
				builder.ArgSpec("SLEEP_FOR_SECONDS", "10"),
			)),
		Check:    PipelineRunHasStateResult(api.ResultSuccess),
		Timeout:  1 * time.Second,
		Expected: "context deadline exceeded",
	}
}

// PipelineRunWrongExpect is a PipelineRunTestBuilder to test Check returning error
func PipelineRunWrongExpect(Namespace string, runID *api.CustomJSON) PipelineRunTest {
	return PipelineRunTest{
		PipelineRun: builder.PipelineRun("wrongexpect-", Namespace,
			builder.PipelineRunSpec(
				builder.LoggingWithRunID(runID),
				builder.JenkinsFileSpec(shared.ExamplePipelineRepoURL,
					"success/Jenkinsfile", shared.ExamplePipelineRepoRevision),
			)),
		Check:    PipelineRunHasStateResult(api.ResultAborted),
		Timeout:  120 * time.Second,
		Expected: `unexpected result: expecting "aborted", got "success"`,
	}
}

// PipelineRunWrongName is a PipelineRunTestBuilder to Check failed pipeline runpipeline run creation
func PipelineRunWrongName(Namespace string, runID *api.CustomJSON) PipelineRunTest {
	return PipelineRunTest{
		PipelineRun: builder.PipelineRun("wrong_Name", Namespace,
			builder.PipelineRunSpec(
				builder.LoggingWithRunID(runID),
				builder.JenkinsFileSpec(shared.ExamplePipelineRepoURL,
					"success/Jenkinsfile", shared.ExamplePipelineRepoRevision),
			)),
		Check:    PipelineRunHasStateResult(api.ResultSuccess),
		Timeout:  120 * time.Second,
		Expected: `pipeline run creation failed: .*wrong_Name.*`,
	}
}

// PipelineRunWithSecretNameConflict is a PipelineRunTestBuilder to test Name conflict with Secrets
func PipelineRunWithSecretNameConflict(Namespace string, runID *api.CustomJSON) PipelineRunTest {
	return PipelineRunTest{
		PipelineRun: builder.PipelineRun("with-secret-name-conflict", Namespace,
			builder.PipelineRunSpec(
				builder.LoggingWithRunID(runID),
				builder.JenkinsFileSpec(shared.ExamplePipelineRepoURL,
					"secret/Jenkinsfile", shared.ExamplePipelineRepoRevision),
			)),
		Check:   PipelineRunHasStateResult(api.ResultSuccess),
		Timeout: 120 * time.Second,
		Secrets: []*v1.Secret{builder.SecretBasicAuth("foo", Namespace, "bar", "baz"),
			builder.SecretBasicAuth("foo", Namespace, "bar", "baz")},
		Expected: `secret creation failed: .*`,
	}
}
