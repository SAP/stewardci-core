// +build e2e

package test

import (
	"testing"

	f "github.com/SAP/stewardci-core/test/framework"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func Test_PipelineRunSingle(t *testing.T) {
	t.Parallel()
	allTests := make([]f.TestPlan, len(AllTestBuilders))
	for i, pipelinerunTestBuilder := range AllTestBuilders {
		allTests[i] = f.TestPlan{TestBuilder: pipelinerunTestBuilder,
			Count: 1,
		}
	}
	f.ExecutePipelineRunTests(t, allTests...)
}

func Test_ClusterWithFinishedPipelines(t *testing.T) {
	tests := []f.TestPlan{
		f.TestPlan{TestBuilder: PipelineRunAbort,
			Count: 20,
		},
	}
	f.ExecutePipelineRunTests(t, tests...)
}
