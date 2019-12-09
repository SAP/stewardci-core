// +build loadtest

package loadtest

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"testing"

	test "github.com/SAP/stewardci-core/test/integrationtest"
	f "github.com/SAP/stewardci-core/test/framework"
)

func Test_ClusterWithFinishedPipelines(t *testing.T) {
	tests := []f.TestPlan{
		f.TestPlan{TestBuilder: test.PipelineRunAbort,
			Count: 100,
		},
	}
	f.ExecutePipelineRunTests(t, tests...)
}
