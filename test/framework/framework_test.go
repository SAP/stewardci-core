// +build frameworktest

package framework

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"testing"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/test/builder"
	v1 "k8s.io/api/core/v1"
)

// frameworkTestBuilders is a list of all test builders
var frameworkTestBuilders = []PipelineRunTestBuilder{
	PipelineRunSleepTooLong,
	PipelineRunWrongExpect,
	PipelineRunWrongName,
	PipelineRunWithSecretNameConflict,
}

const pipelineRepoURL = "https://github.com/SAP-samples/stewardci-example-pipelines"

func Test_FirstFinishBeforeSecondStarts(t *testing.T) {
	test := TestPlan{TestBuilder: PipelineRunWrongName,
		Count:         2,
		CreationDelay: time.Second * 1,
		Name:          "FirstFinishBeforeSecondStartsu",
	}
	ctx := setup(t)
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
	ctx := setup(t)
	executePipelineRunTests(ctx, t, allTests...)
}

// PipelineRunSleepTooLong is a PipelineRunTestBuilder to test if Timeout works correctly
func PipelineRunSleepTooLong(Namespace string) PipelineRunTest {
	return PipelineRunTest{
		PipelineRun: builder.PipelineRun("sleeptoolong-", Namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURL,
					"sleep/Jenkinsfile"),
				builder.ArgSpec("SLEEP_FOR_SECONDS", "10"),
			)),
		Check:    PipelineRunHasStateResult(api.ResultSuccess),
		Timeout:  1 * time.Second,
		Expected: "context deadline exceeded",
	}
}

// PipelineRunWrongExpect is a PipelineRunTestBuilder to test Check returning error
func PipelineRunWrongExpect(Namespace string) PipelineRunTest {
	return PipelineRunTest{
		PipelineRun: builder.PipelineRun("wrongexpect-", Namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURL,
					"success/Jenkinsfile"),
			)),
		Check:    PipelineRunHasStateResult(api.ResultAborted),
		Timeout:  120 * time.Second,
		Expected: `unexpected result: expecting "aborted", got "success"`,
	}
}

// PipelineRunWrongName is a PipelineRunTestBuilder to Check failed pipeline runpipeline run creation
func PipelineRunWrongName(Namespace string) PipelineRunTest {
	return PipelineRunTest{
		PipelineRun: builder.PipelineRun("wrong_Name", Namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURL,
					"success/Jenkinsfile"),
			)),
		Check:    PipelineRunHasStateResult(api.ResultSuccess),
		Timeout:  120 * time.Second,
		Expected: `pipeline run creation failed: .*wrong_Name.*`,
	}
}

// PipelineRunWithSecretNameConflict is a PipelineRunTestBuilder to test Name conflict with Secrets
func PipelineRunWithSecretNameConflict(Namespace string) PipelineRunTest {
	return PipelineRunTest{
		PipelineRun: builder.PipelineRun("with-secret-name-conflict", Namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURL,
					"secret/Jenkinsfile"),
			)),
		Check:   PipelineRunHasStateResult(api.ResultSuccess),
		Timeout: 120 * time.Second,
		Secrets: []*v1.Secret{builder.SecretBasicAuth("foo", Namespace, "bar", "baz"),
			builder.SecretBasicAuth("foo", Namespace, "bar", "baz")},
		Expected: `secret creation failed: .*`,
	}
}
