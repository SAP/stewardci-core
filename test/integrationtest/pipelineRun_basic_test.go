//go:build e2eb
// +build e2eb

package integrationtest

import (
	"testing"
	"time"

	f "github.com/SAP/stewardci-core/test/framework"
)

func Test_PipelineRunSingle_Basic(t *testing.T) {
	t.Parallel()
	allTests := make([]f.TestPlan, len(BasicTestBuilders))
	for i, pipelinerunTestBuilder := range BasicTestBuilders {
		allTests[i] = f.TestPlan{TestBuilder: pipelinerunTestBuilder,
			Count:         1,
			CreationDelay: time.Second * 1,
		}
	}
	f.ExecutePipelineRunTests(t, allTests...)
}
