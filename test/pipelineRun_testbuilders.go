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
	//        PipelineRunSleepTooLong,
	PipelineRunFail,
	PipelineRunOK,
}

// PipelineRunSleep is a pipelineRunTestBuilder to build pipelineRunTest which sleeps for one second
func PipelineRunSleep(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "sleep",
		pipelineRun: builder.PipelineRun(namespace,
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
		pipelineRun: builder.PipelineRun(namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/sap-production/demo-pipelines",
					"master",
					"sleep/Jenkinsfile"),
				builder.ArgSpec("SLEEP_FOR_SECONDS", "120"),
			)),
		check:   PipelineRunHasStateResult(api.ResultSuccess),
		timeout: 100 * time.Second,
	}
}

// PipelineRunFail is a pipelineRunTestBuilder to build pipelineRunTest which fails
func PipelineRunFail(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "error",
		pipelineRun: builder.PipelineRun(namespace,
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
		pipelineRun: builder.PipelineRun(namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/sap-production/demo-pipelines",
					"master",
					"success/Jenkinsfile"),
			)),
		check:   PipelineRunHasStateResult(api.ResultSuccess),
		timeout: 120 * time.Second,
	}
}
