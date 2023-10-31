//go:build closednet

package networkpolicy

import (
	"testing"

	f "github.com/SAP/stewardci-core/test/framework"
)

func Test_PipelineRunClosedNetworkPolicy(t *testing.T) {
	npTest := f.TestPlan{TestBuilder: PipelineRunNetworkClosedPolicy,
		Count: 1,
	}
	f.ExecutePipelineRunTests(t, npTest)
}
