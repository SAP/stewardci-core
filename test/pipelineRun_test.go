// +build e2e

package test

import (
	"testing"
)

func Test_PipelineRunSingle(t *testing.T) {
	t.Parallel()
        allTests := make([]testPlan, len(AllTestBuilders))
        for i, pipelinerunTestBuilder := range AllTestBuilders {
           allTests[i] = testPlan{testBuilder: pipelinerunTestBuilder,
                                  parallel: 1,
           }
        }
        executePipelineRunTests(t,allTests...)
}


