// +build e2e

package integrationtest

import (
	"testing"

	test "github.com/SAP/stewardci-core/test"
	f "github.com/SAP/stewardci-core/test/frameworkth/gcp"
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
