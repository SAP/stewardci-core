// +build loadtest

package test

import (
	"testing"
)

func Test_PipelineRuns(t *testing.T) {
	executePipelineRunTests(t,
		testPlan{testBuilder: PipelineRunSleep,
			parallel: 3,
		},
		testPlan{testBuilder: PipelineRunFail,
			parallel: 3,
		},
		testPlan{testBuilder: PipelineRunOK,
			parallel: 3,
		},
	)
}

