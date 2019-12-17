package integrationtest

import (
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	builder "github.com/SAP/stewardci-core/test/builder"
	f "github.com/SAP/stewardci-core/test/framework"
	v1 "k8s.io/api/core/v1"
)

// AllTestBuilders is a list of all test builders
var AllTestBuilders = []f.PipelineRunTestBuilder{
	PipelineRunAbort,
	PipelineRunSleep,
	PipelineRunFail,
	PipelineRunOK,
	PipelineRunWithSecret,
	PipelineRunWrongJenkinsfileRepo,
	PipelineRunWrongJenkinsfilePath,
	PipelineRunWrongJenkinsfileRepoWithUser,
}

const pipelineRepoURL = "https://github.com/SAP-samples/stewardci-example-pipelines"

// PipelineRunAbort is a PipelineRunTestBuilder to build a PipelineRunTest with aborted pipeline
func PipelineRunAbort(Namespace string) f.PipelineRunTest {
	return f.PipelineRunTest{
		PipelineRun: builder.PipelineRun("sleep-", Namespace,
			builder.PipelineRunSpec(
				builder.Abort(),
				builder.JenkinsFileSpec(pipelineRepoURL,
					"sleep/Jenkinsfile"),
			)),
		Check:   f.PipelineRunHasStateResult(api.ResultAborted),
		Timeout: 15 * time.Second,
	}

}

// PipelineRunSleep is a PipelineRunTestBuilder to build PipelineRunTest which sleeps for one second
func PipelineRunSleep(Namespace string) f.PipelineRunTest {
	return f.PipelineRunTest{
		PipelineRun: builder.PipelineRun("sleep-", Namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURL,
					"sleep/Jenkinsfile"),
				builder.ArgSpec("SLEEP_FOR_SECONDS", "1"),
			)),
		Check:   f.PipelineRunHasStateResult(api.ResultSuccess),
		Timeout: 600 * time.Second,
	}
}

// PipelineRunFail is a PipelineRunTestBuilder to build PipelineRunTest which fails
func PipelineRunFail(Namespace string) f.PipelineRunTest {
	return f.PipelineRunTest{
		PipelineRun: builder.PipelineRun("error-", Namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURL,
					"error/Jenkinsfile"),
			)),
		Check:   f.PipelineRunHasStateResult(api.ResultErrorContent),
		Timeout: 600 * time.Second,
	}
}

// PipelineRunOK is a PipelineRunTestBuilder to build PipelineRunTest which succeeds
func PipelineRunOK(Namespace string) f.PipelineRunTest {
	return f.PipelineRunTest{
		PipelineRun: builder.PipelineRun("ok-", Namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURL,
					"success/Jenkinsfile"),
			)),
		Check:   f.PipelineRunHasStateResult(api.ResultSuccess),
		Timeout: 600 * time.Second,
	}
}

// PipelineRunWithSecret is a PipelineRunTestBuilder to build PipelineRunTest which uses Secrets
func PipelineRunWithSecret(Namespace string) f.PipelineRunTest {
	return f.PipelineRunTest{
		PipelineRun: builder.PipelineRun("with-secret-", Namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURL,
					"secret/Jenkinsfile"),
				builder.ArgSpec("SECRETID", "with-secret-foo"),
				builder.ArgSpec("EXPECTEDUSER", "bar"),
				builder.ArgSpec("EXPECTEDPWD", "baz"),
				builder.Secret("with-secret-foo"),
			)),
		Check:   f.PipelineRunHasStateResult(api.ResultSuccess),
		Timeout: 120 * time.Second,
		Secrets: []*v1.Secret{builder.SecretBasicAuth("with-secret-foo", Namespace, "bar", "baz")},
	}
}

// PipelineRunMissingSecret is a PipelineRunTestBuilder to build PipelineRunTest which uses Secrets
func PipelineRunMissingSecret(Namespace string) f.PipelineRunTest {
	return f.PipelineRunTest{
		PipelineRun: builder.PipelineRun("missing-secret-", Namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURL,
					"secret/Jenkinsfile"),
				builder.ArgSpec("SECRETID", "foo"),
				builder.ArgSpec("EXPECTEDUSER", "bar"),
				builder.ArgSpec("EXPECTEDPWD", "baz"),
			)),
		Check:   f.PipelineRunHasStateResult(api.ResultErrorContent),
		Timeout: 120 * time.Second,
		Secrets: []*v1.Secret{builder.SecretBasicAuth("missing-secret-foo", Namespace, "bar", "baz")},
	}
}

// PipelineRunWrongJenkinsfileRepo is a PipelineRunTestBuilder to build PipelineRunTest with wrong jenkinsfile repo url
func PipelineRunWrongJenkinsfileRepo(Namespace string) f.PipelineRunTest {
	return f.PipelineRunTest{
		PipelineRun: builder.PipelineRun("wrong-jenkinsfile-repo-", Namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/SAP/steward-foo",
					"Jenkinsfile"),
			)),
		Check: f.PipelineRunMessageOnFinished(`Command ['git' 'clone' 'https://github.com/SAP/steward-foo' '.'] failed with exit code 128
Error output:
Cloning into '.'...
fatal: could not read Username for 'https://github.com': No such device or address`),
		Timeout: 120 * time.Second,
	}
}

// PipelineRunWrongJenkinsfileRepoWithUser is a PipelineRunTestBuilder to build PipelineRunTest with wrong jenkinsfile repo url
func PipelineRunWrongJenkinsfileRepoWithUser(Namespace string) f.PipelineRunTest {
	return f.PipelineRunTest{
		PipelineRun: builder.PipelineRun("wrong-jenkinsfile-repo-user-", Namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/SAP/steward-foo",
					"Jenkinsfile",
					builder.RepoAuthSecret("repo-auth"),
				),
			)),
		Secrets: []*v1.Secret{builder.SecretBasicAuth("repo-auth", Namespace, "bar", "baz")},
		Check: f.PipelineRunMessageOnFinished(`Command ['git' 'clone' 'https://github.com/SAP/steward-foo' '.'] failed with exit code 128
Error output:
Cloning into '.'...
fatal: could not read Username for 'https://github.com': No such device or address`),
		Timeout: 120 * time.Second,
	}
}

// PipelineRunWrongJenkinsfilePath is a PipelineRunTestBuilder to build PipelineRunTest with wrong jenkinsfile path
func PipelineRunWrongJenkinsfilePath(Namespace string) f.PipelineRunTest {
	return f.PipelineRunTest{
		PipelineRun: builder.PipelineRun("wrong-jenkinsfile-path-", Namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURL,
					"not_existing_path/Jenkinsfile"),
			)),
		Check: f.PipelineRunMessageOnFinished(`Command ['/app/bin/jenkinsfile-runner' '-w' '/app/jenkins' '-p' '/usr/share/jenkins/ref/plugins' '--runHome' '/jenkins_home' '--no-sandbox' '--build-number' '1' '-f' 'not_existing_path/Jenkinsfile'] failed with exit code 255
Error output:
no Jenkinsfile in current directory.`),
		Timeout: 120 * time.Second,
	}
}
