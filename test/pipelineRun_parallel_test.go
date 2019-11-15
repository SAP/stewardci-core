// +build loadtest

package test

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
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
