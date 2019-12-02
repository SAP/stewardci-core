// +build e2e

package test

import (
	"testing"
"time"
"log"

	f "github.com/SAP/stewardci-core/test/framework"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
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

func Test_ClusterWithFinishedPipelines(t *testing.T) {
	for round := 0; round < 5; round++ {
		log.Printf("Round %d", round)
		tests := []f.TestPlan{
			f.TestPlan{TestBuilder: PipelineRunOK,
				Count:         2,
				CreationDelay: time.Second * 10,
			},
			f.TestPlan{TestBuilder: PipelineRunWrongJenkinsfileRepo,
				Count: 10,
			},
		}
  f.ExecutePipelineRunTests(t, tests...)
 	}
}
