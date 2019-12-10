// +build loadtest

package loadtest

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"testing"

	f "github.com/SAP/stewardci-core/test/framework"
	test "github.com/SAP/stewardci-core/test/integrationtest"
)

func Test_ClusterWithFinishedPipelines(t *testing.T) {
	tests := []f.TestPlan{
		f.TestPlan{TestBuilder: test.PipelineRunAbort,
			Count: 100,
		},
	}
	f.ExecutePipelineRunTests(t, tests...)
}
