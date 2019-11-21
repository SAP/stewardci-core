package test

import (
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/test/builder"
	v1 "k8s.io/api/core/v1"
)

// PipelineRunTest is a test for a pipeline run
type PipelineRunTest struct {
	name        string
	pipelineRun *api.PipelineRun
	secrets     []*v1.Secret
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
	PipelineRunWithSecret,
	PipelineRunMissingSecret,
	PipelineRunWrongJenkinsfileRepo,
	PipelineRunWrongJenkinsfilePath,
	PipelineRunWrongJenkinsfileRepoWithUser,
}

const pipelineRepoURL = "https://github.com/SAP-samples/stewardci-example-pipelines"
const pipelineRepoURLrinckm = "https://github.com/rinckm/stewardci-example-pipelines"

// PipelineRunSleep is a pipelineRunTestBuilder to build pipelineRunTest which sleeps for one second
func PipelineRunSleep(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "sleep",
		pipelineRun: builder.PipelineRun("sleep-", namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURL,
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
				builder.JenkinsFileSpec(pipelineRepoURL,
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
				builder.JenkinsFileSpec(pipelineRepoURL,
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
				builder.JenkinsFileSpec(pipelineRepoURL,
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
				builder.JenkinsFileSpec(pipelineRepoURL,
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
				builder.JenkinsFileSpec(pipelineRepoURL,
					"success/Jenkinsfile"),
			)),
		check:    PipelineRunHasStateResult(api.ResultSuccess),
		timeout:  120 * time.Second,
		expected: `pipeline run creation failed: .*wrong_name.*`,
	}
}

// PipelineRunWithSecret is a pipelineRunTestBuilder to build pipelineRunTest which uses secrets
func PipelineRunWithSecret(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "with-secret",
		pipelineRun: builder.PipelineRun("with-secret-", namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURLrinckm,
					"secret/Jenkinsfile"),
				builder.ArgSpec("SECRETID", "with-secret-foo"),
				builder.ArgSpec("EXPECTEDUSER", "bar"),
				builder.ArgSpec("EXPECTEDPWD", "baz"),
				builder.Secret("with-secret-foo"),
			)),
		check:   PipelineRunHasStateResult(api.ResultSuccess),
		timeout: 120 * time.Second,
		secrets: []*v1.Secret{builder.SecretBasicAuth("with-secret-foo", namespace, "bar", "baz")},
	}
}

// PipelineRunMissingSecret is a pipelineRunTestBuilder to build pipelineRunTest which uses secrets
func PipelineRunMissingSecret(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "missing-secret",
		pipelineRun: builder.PipelineRun("missing-secret-", namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURLrinckm,
					"secret/Jenkinsfile"),
				builder.ArgSpec("SECRETID", "foo"),
				builder.ArgSpec("EXPECTEDUSER", "bar"),
				builder.ArgSpec("EXPECTEDPWD", "baz"),
			)),
		check:   PipelineRunHasStateResult(api.ResultErrorContent),
		timeout: 120 * time.Second,
		secrets: []*v1.Secret{builder.SecretBasicAuth("missing-secret-foo", namespace, "bar", "baz")},
	}
}

// PipelineRunWrongJenkinsfileRepo is a pipelineRunTestBuilder to build pipelineRunTest with wrong jenkinsfile repo url
func PipelineRunWrongJenkinsfileRepo(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "wrong-jenkinsfile-repo",
		pipelineRun: builder.PipelineRun("wrong-jenkinsfile-repo-", namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/SAP/steward-foo",
					"Jenkinsfile"),
			)),
		check: PipelineRunMessageOnFinished(`Command ['git' 'clone' 'https://github.com/SAP/steward-foo' '.'] failed with exit code 128
Error output:
Cloning into '.'...
fatal: could not read Username for 'https://github.com': No such device or address`),
		timeout: 120 * time.Second,
	}
}

// PipelineRunWrongJenkinsfileRepoWithUser is a pipelineRunTestBuilder to build pipelineRunTest with wrong jenkinsfile repo url
func PipelineRunWrongJenkinsfileRepoWithUser(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "wrong-jenkinsfile-repo-user",
		pipelineRun: builder.PipelineRun("wrong-jenkinsfile-repo-user-", namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/SAP/steward-foo",
					"Jenkinsfile",
					builder.RepoAuthSecret("repo-auth"),
				),
			)),
		secrets: []*v1.Secret{builder.SecretBasicAuth("repo-auth", namespace, "bar", "baz")},
		check: PipelineRunMessageOnFinished(`Command ['git' 'clone' 'https://github.com/SAP/steward-foo' '.'] failed with exit code 128
Error output:
Cloning into '.'...
fatal: could not read Username for 'https://github.com': No such device or address`),
		timeout: 120 * time.Second,
	}
}

// PipelineRunWrongJenkinsfilePath is a pipelineRunTestBuilder to build pipelineRunTest with wrong jenkinsfile path
func PipelineRunWrongJenkinsfilePath(namespace string) PipelineRunTest {
	return PipelineRunTest{
		name: "wrong-jenkinsfile-path",
		pipelineRun: builder.PipelineRun("wrong-jenkinsfile-path-", namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec(pipelineRepoURL,
					"not_existing_path/Jenkinsfile"),
			)),
		check: PipelineRunMessageOnFinished(`Command ['/app/bin/jenkinsfile-runner' '-w' '/app/jenkins' '-p' '/usr/share/jenkins/ref/plugins' '--runHome' '/jenkins_home' '--no-sandbox' '--build-number' '1' '-f' 'not_existing_path/Jenkinsfile'] failed with exit code 255
Error output:
no Jenkinsfile in current directory.`),
		timeout: 120 * time.Second,
	}
}
