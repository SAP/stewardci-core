package test

import (
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/test/builder"
)

// PipelineRunTest is a test for a pipeline run
type PipelineRunTest struct {
	name        string
	pipelineRun *api.PipelineRun
	check       PipelineRunCheck
	expected    string
	timeout     time.Duration
}

// PipelineRunTestBuilder is a funciton creating a PipelineRunTest for a defined namespace
type PipelineRunTestBuilder = func(string) PipelineRunTest

type testPlan struct {
	testBuilder      PipelineRunTestBuilder
	parallel         int
	parallelCreation bool
	creationDelay    time.Duration
}

// AllTestBuilders is a list of all test builders
var AllTestBuilders = []PipelineRunTestBuilder{
	PipelineRunSleep,
	PipelineRunSleepTooLong,
	PipelineRunFail,
	PipelineRunOK,
	PipelineRunWrongExpect,
	PipelineRunWrongName,
}

// PipelineRunSleep is a pipelineRunTestBuilder to build pipelineRunTest which sleeps for one second
func PipelineRunSleep(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "sleep",
		pipelineRun: builder.PipelineRun("sleep-", namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/sap-production/demo-pipelines",
					"master",
					"sleep/Jenkinsfile"),
				builder.ArgSpec("SLEEP_FOR_SECONDS", "1"),
			)),
		check:   PipelineRunHasStateResult(api.ResultSuccess),
		timeout: 120 * time.Second,
	}
}

// PipelineRunSleepTooLong is a pipelineRunTestBuilder to build pipelineRunTest which sleeps for one second
func PipelineRunSleepTooLong(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "sleep_too_long",
		pipelineRun: builder.PipelineRun("sleeptoolong-", namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/sap-production/demo-pipelines",
					"master",
					"sleep/Jenkinsfile"),
				builder.ArgSpec("SLEEP_FOR_SECONDS", "10"),
			)),
		check:    PipelineRunHasStateResult(api.ResultSuccess),
		timeout:  1 * time.Second,
		expected: "context deadline exceeded",
	}
}

// PipelineRunFail is a pipelineRunTestBuilder to build pipelineRunTest which fails
func PipelineRunFail(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "error",
		pipelineRun: builder.PipelineRun("error-", namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/sap-production/demo-pipelines",
					"master",
					"error/Jenkinsfile"),
			)),
		check:   PipelineRunHasStateResult(api.ResultErrorContent),
		timeout: 120 * time.Second,
	}
}

// PipelineRunOK is a pipelineRunTestBuilder to build pipelineRunTest which succeeds
func PipelineRunOK(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "ok",
		pipelineRun: builder.PipelineRun("ok-", namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/sap-production/demo-pipelines",
					"master",
					"success/Jenkinsfile"),
			)),
		check:   PipelineRunHasStateResult(api.ResultSuccess),
		timeout: 120 * time.Second,
	}
}

// PipelineRunWrongExpect is a pipelineRunTestBuilder to build pipelineRunTest which succeeds but test expects wrong result
func PipelineRunWrongExpect(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "wrong_expect",
		pipelineRun: builder.PipelineRun("wrongexpect-", namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/sap-production/demo-pipelines",
					"master",
					"success/Jenkinsfile"),
			)),
		check:    PipelineRunHasStateResult(api.ResultKilled),
		timeout:  120 * time.Second,
		expected: `Unexpected result: expecting "killed", got "success"`,
	}
}

// PipelineRunWrongName is a pipelineRunTestBuilder to build pipelineRunTest which wrong name
func PipelineRunWrongName(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "wrong_name--",
		pipelineRun: builder.PipelineRun("wrong_name", namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/sap-production/demo-pipelines",
					"master",
					"success/Jenkinsfile"),
			)),
		check:    PipelineRunHasStateResult(api.ResultSuccess),
		timeout:  120 * time.Second,
		expected: `pipeline run creation failed: .*wrong_name.*`,
	}
}
