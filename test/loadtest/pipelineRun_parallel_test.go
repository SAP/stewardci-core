// +build loadtest

package test

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"testing"
	"time"

	"github.com/SAP/stewardci-core/test"
	f "github.com/SAP/stewardci-core/test/framework"
)

func Test_PipelineRuns_delayedCreation(t *testing.T) {
	f.ExecutePipelineRunTests(t,
		f.TestPlan{TestBuilder: test.PipelineRunOK,
			Parallel:      10,
			CreationDelay: time.Duration(5 * time.Second),
		},
	)
}

func Test_PipelineRuns(t *testing.T) {
	f.ExecutePipelineRunTests(t,
		f.TestPlan{TestBuilder: test.PipelineRunSleep,
			Parallel: 3,
		},
		f.TestPlan{TestBuilder: test.PipelineRunFail,
			Parallel: 3,
		},
		f.TestPlan{TestBuilder: test.PipelineRunOK,
			Parallel: 3,
		},
	)
}

func Test_PipelineRuns_ParallelCreation(t *testing.T) {
	f.ExecutePipelineRunTests(t,
		f.TestPlan{TestBuilder: test.PipelineRunSleep,
			Parallel:         3,
			ParallelCreation: true,
		},
		f.TestPlan{TestBuilder: test.PipelineRunFail,
			Parallel:         3,
			ParallelCreation: true,
		},
		f.TestPlan{TestBuilder: test.PipelineRunOK,
			Parallel:         3,
			ParallelCreation: true,
		},
	)
}

func Test_PipelineRuns_CreationDelay(t *testing.T) {
	f.ExecutePipelineRunTests(t,
		f.TestPlan{TestBuilder: test.PipelineRunSleep,
			Parallel:      3,
			CreationDelay: time.Duration(1 * time.Second),
		},
		f.TestPlan{TestBuilder: test.PipelineRunFail,
			Parallel:      3,
			CreationDelay: time.Duration(1 * time.Second),
		},
		f.TestPlan{TestBuilder: test.PipelineRunOK,
			Parallel:      3,
			CreationDelay: time.Duration(1 * time.Second),
		},
	)
}
