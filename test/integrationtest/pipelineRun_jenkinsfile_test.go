//go:build e2ej
// +build e2ej

package integrationtest

import (
	"testing"
	"time"

	f "github.com/SAP/stewardci-core/test/framework"
)

func Test_PipelineRunSingle_Jenkinsfile(t *testing.T) {
	t.Parallel()
	allTests := make([]f.TestPlan, len(JenkinsfileTestBuilders))
	for i, pipelinerunTestBuilder := range JenkinsfileTestBuilders {
		allTests[i] = f.TestPlan{TestBuilder: pipelinerunTestBuilder,
			Count:         1,
			CreationDelay: time.Second * 1,
		}
	}
	f.ExecutePipelineRunTests(t, allTests...)
}
