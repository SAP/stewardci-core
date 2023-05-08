//go:build e2es
// +build e2es

package integrationtest

import (
	"testing"
	"time"

	f "github.com/SAP/stewardci-core/test/framework"
)

func Test_PipelineRunSingle_Secrets(t *testing.T) {
	t.Parallel()
	allTests := make([]f.TestPlan, len(SecretTestBuilders))
	for i, pipelinerunTestBuilder := range SecretTestBuilders {
		allTests[i] = f.TestPlan{TestBuilder: pipelinerunTestBuilder,
			Count:         1,
			CreationDelay: time.Second * 1,
		}
	}
	f.ExecutePipelineRunTests(t, allTests...)
}
