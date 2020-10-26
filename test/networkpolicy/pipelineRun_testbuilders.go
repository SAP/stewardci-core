package networkpolicy

import (
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	builder "github.com/SAP/stewardci-core/test/builder"
	f "github.com/SAP/stewardci-core/test/framework"
)

const pipelineRepoURL = "https://github.com/SAP-samples/stewardci-example-pipelines"

// PipelineRunNetworkClosedPolicy is a PipelineRunTestBuilder to build a PipelineRunTest to check network policy
func PipelineRunNetworkClosedPolicy(Namespace string, runID *api.CustomJSON) f.PipelineRunTest {
	return f.PipelineRunTest{
		PipelineRun: builder.PipelineRun("net-", Namespace,
			builder.PipelineRunSpec(
				builder.LoggingWithRunID(runID),
				builder.JenkinsFileSpec(pipelineRepoURL,
					"netcat/Jenkinsfile"),
			)),
		Check:   f.PipelineRunHasStateResult(api.ResultErrorContent),
		Timeout: 600 * time.Second,
	}
}

// PipelineRunNetworkOpenPolicy is a PipelineRunTestBuilder to build a PipelineRunTest to check network policy
func PipelineRunNetworkOpenPolicy(Namespace string, runID *api.CustomJSON) f.PipelineRunTest {
	return f.PipelineRunTest{
		PipelineRun: builder.PipelineRun("net-", Namespace,
			builder.PipelineRunSpec(
				builder.LoggingWithRunID(runID),
				builder.JenkinsFileSpec(pipelineRepoURL,
					"netcat/Jenkinsfile"),
			)),
		Check:   f.PipelineRunHasStateResult(api.ResultSuccess),
		Timeout: 600 * time.Second,
	}
}
