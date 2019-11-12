// +build e2e

package test

import (
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/SAP/stewardci-core/test/builder"
	"gotest.tools/assert"
)

type pipelineRunTest struct {
	name        string
	pipelineRun *api.PipelineRun
	check       PipelineRunCheck
}

type pipelineRunTestBuilder = func(string) pipelineRunTest

func PipelineRunSleep(namespace string) pipelineRunTest {
	return pipelineRunTest{
		name: "sleep",
		pipelineRun: builder.PipelineRun(namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/sap-production/demo-pipelines",
					"master",
					"sleep/Jenkinsfile"),
				builder.ArgSpec("SLEEP_FOR_SECONDS", "1"),
			)),
		check: PipelineRunHasStateResult(api.ResultSuccess),
	}
}

func PipelineRunFail(namespace string) pipelineRunTest {
	return pipelineRunTest{
		name: "error",
		pipelineRun: builder.PipelineRun(namespace,
			builder.PipelineRunSpec(
				builder.JenkinsFileSpec("https://github.com/sap-production/demo-pipelines",
					"master",
					"error/Jenkinsfile"),
			)),
		check: PipelineRunHasStateResult(api.ResultErrorContent),
	}
}

func TestPipelineRuns(t *testing.T) {
	executePipelineRunTests(t, PipelineRunSleep, PipelineRunFail)
}

func executePipelineRunTests(t *testing.T, testBuilders ...pipelineRunTestBuilder) {
	t.Parallel()
	clientFactory, namespace, waiter := setup(t)
	test := TenantSuccessTest(namespace)
	tenant := test.tenant
	tenant, err := CreateTenant(clientFactory, tenant)
	assert.NilError(t, err)

	defer DeleteTenant(clientFactory, tenant)
	check := CreateTenantCondition(tenant, test.check, test.name)
	err = waiter.WaitFor(check)
	assert.NilError(t, err)
	tenant, err = GetTenant(clientFactory, tenant)
	assert.NilError(t, err)
	tnn := tenant.Status.TenantNamespaceName
	for _, testBuilder := range testBuilders {
		pipelineTest := testBuilder(tnn)
		t.Run(pipelineTest.name, func(t *testing.T) {
			pipelineTest := pipelineTest
			t.Parallel()
			pr, err := createPipelineRun(clientFactory, pipelineTest.pipelineRun)
			assert.NilError(t, err)

			pipelineRunCheck := CreatePipelineRunCondition(pr, pipelineTest.check, pipelineTest.name)
			err = waiter.WaitFor(pipelineRunCheck)
			assert.NilError(t, err)
		})
	}
}

func createPipelineRun(clientFactory k8s.ClientFactory, pipelineRun *api.PipelineRun) (*api.PipelineRun, error) {
	stewardClient := clientFactory.StewardV1alpha1().PipelineRuns(pipelineRun.GetNamespace())
	return stewardClient.Create(pipelineRun)
}
