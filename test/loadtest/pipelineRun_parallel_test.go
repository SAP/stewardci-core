//go:build loadtest
// +build loadtest

package loadtest

import (
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"testing"
	"time"

	f "github.com/SAP/stewardci-core/test/framework"
	test "github.com/SAP/stewardci-core/test/integrationtest"
)

func Test_Loadtest_delay(t *testing.T) {
	var delays = []time.Duration{
		time.Duration(5 * time.Second),
		time.Duration(2 * time.Second),
		time.Duration(1 * time.Second),
	}

	var tests = make([]f.TestPlan, len(delays))
	for i, delay := range delays {
		tests[i] =
			f.TestPlan{
				TestBuilder:   test.PipelineRunOK,
				Count:         300,
				CreationDelay: delay,
			}
	}
	f.ExecutePipelineRunTests(t, tests...)
}

func Test_Loadtest_ok(t *testing.T) {
	f.ExecutePipelineRunTests(t,
		f.TestPlan{
			TestBuilder: test.PipelineRunOK,
			Count:       2,
		},
		f.TestPlan{
			TestBuilder:      test.PipelineRunOK,
			Count:            2,
			ParallelCreation: true,
		},
		f.TestPlan{
			TestBuilder:   test.PipelineRunOK,
			Count:         2,
			CreationDelay: time.Duration(5 * time.Second),
		},

		f.TestPlan{
			TestBuilder: test.PipelineRunOK,
			Count:       10,
		},
		f.TestPlan{
			TestBuilder:      test.PipelineRunOK,
			Count:            10,
			ParallelCreation: true,
		},
		f.TestPlan{
			TestBuilder:   test.PipelineRunOK,
			Count:         10,
			CreationDelay: time.Duration(5 * time.Second),
		},

		f.TestPlan{
			TestBuilder: test.PipelineRunOK,
			Count:       100,
		},
		f.TestPlan{
			TestBuilder:      test.PipelineRunOK,
			Count:            100,
			ParallelCreation: true,
		},
		f.TestPlan{
			TestBuilder:   test.PipelineRunOK,
			Count:         100,
			CreationDelay: time.Duration(5 * time.Second),
		},
	)
}
