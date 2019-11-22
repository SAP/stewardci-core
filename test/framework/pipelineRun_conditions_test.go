// +build xxx

package framework

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"github.com/SAP/stewardci-core/test/builder"
	"gotest.tools/assert"
	"testing"
	"time"
)

func Test_PipelineCondition(t *testing.T) {
	Check := PipelineRunHasStateResult(api.ResultSuccess)
	PipelineRun :=
		builder.PipelineRun("Namespace1")
	clientFactory := fake.NewClientFactory()
	ctx := context.Background()
	ctx.SetClientFactory(clientFactroy)
	pr, err := createPipelineRun(clientFactory, PipelineRun)
	assert.NilError(t, err)
	PipelineRunCheck := CreatePipelineRunCondition(pr, Check, "Test")
	errorChan := make(chan error)

	go func() {
		errorChan <- WaitFor(ctx, PipelineRunCheck)
	}()
	time.Sleep(3 * time.Second)
	setState(clientFactory, pr, api.ResultSuccess)
	err = <-errorChan
	assert.NilError(t, err)
}
