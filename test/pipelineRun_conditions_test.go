// +build xxx

package test

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"github.com/SAP/stewardci-core/test/builder"
	"gotest.tools/assert"
	"testing"
	"time"
)

func Test_PipelineCondition(t *testing.T) {
	check := PipelineRunHasStateResult(api.ResultSuccess)
	pipelineRun :=
		builder.PipelineRun("namespace1")
	clientFactory := fake.NewClientFactory()
	ctx := context.Background()
	ctx.SetClientFactory(clientFactroy)
	pr, err := createPipelineRun(clientFactory, pipelineRun)
	assert.NilError(t, err)
	pipelineRunCheck := CreatePipelineRunCondition(pr, check, "Test")
	errorChan := make(chan error)

	go func() {
		errorChan <- WaitFor(ctx, pipelineRunCheck)
	}()
	time.Sleep(3 * time.Second)
	setState(clientFactory, pr, api.ResultSuccess)
	err = <-errorChan
	assert.NilError(t, err)
}
