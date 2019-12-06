// +build loadtest

package test

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"testing"

	"github.com/SAP/stewardci-core/test"
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
