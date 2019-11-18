// +build loadtest

package test

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"testing"
	"time"
)

func Test_PipelineRuns_delayedCreation(t *testing.T) {
	executePipelineRunTests(t,
		testPlan{testBuilder: PipelineRunOK,
			parallel:      10,
			creationDelay: time.Duration(5 * time.Second),
		},
	)
}

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

func Test_PipelineRuns_parallelCreation(t *testing.T) {
	executePipelineRunTests(t,
		testPlan{testBuilder: PipelineRunSleep,
			parallel:         3,
			parallelCreation: true,
		},
		testPlan{testBuilder: PipelineRunFail,
			parallel:         3,
			parallelCreation: true,
		},
		testPlan{testBuilder: PipelineRunOK,
			parallel:         3,
			parallelCreation: true,
		},
	)
}

func Test_PipelineRuns_creationDelay(t *testing.T) {
	executePipelineRunTests(t,
		testPlan{testBuilder: PipelineRunSleep,
			parallel:      3,
			creationDelay: time.Duration(1 * time.Second),
		},
		testPlan{testBuilder: PipelineRunFail,
			parallel:      3,
			creationDelay: time.Duration(1 * time.Second),
		},
		testPlan{testBuilder: PipelineRunOK,
			parallel:      3,
			creationDelay: time.Duration(1 * time.Second),
		},
	)
}
