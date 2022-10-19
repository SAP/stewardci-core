//go:build loadtest
// +build loadtest

package loadtest

import (
	"testing"
	"time"

	f "github.com/SAP/stewardci-core/test/framework"
	test "github.com/SAP/stewardci-core/test/integrationtest"
)

func Test_ClusterWithFinishedPipelines(t *testing.T) {
	tests := []f.TestPlan{
		f.TestPlan{TestBuilder: test.PipelineRunAbort,
			Count:         100,
			CreationDelay: time.Duration(1 * time.Second),
		},
	}
	f.ExecutePipelineRunTests(t, tests...)
}

func Test_ClusterWithWrongUrl(t *testing.T) {
	tests := []f.TestPlan{
		f.TestPlan{TestBuilder: test.PipelineRunWrongJenkinsfileRepo,
			Count:         1000,
			CreationDelay: time.Duration(1 * time.Second),
		},
	}
	f.ExecutePipelineRunTests(t, tests...)
}
