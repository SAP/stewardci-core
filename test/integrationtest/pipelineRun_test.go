//go:build e2e

package integrationtest

import (
	f "github.com/SAP/stewardci-core/test/framework"
	"testing"
	"time"
)

func Test_PipelineRunAll(t *testing.T) {
	t.Parallel()
	allTests := make([]f.TestPlan, len(AllTestBuilders))
	for i, pipelinerunTestBuilder := range AllTestBuilders {
		allTests[i] = f.TestPlan{TestBuilder: pipelinerunTestBuilder,
			Count: 1,
		}
	}
	f.ExecutePipelineRunTests(t, allTests...)
}

func Test_PipelineRunSingle_Basic(t *testing.T) {
	allTests := make([]f.TestPlan, len(BasicTestBuilders))
	for i, pipelinerunTestBuilder := range BasicTestBuilders {
		allTests[i] = f.TestPlan{TestBuilder: pipelinerunTestBuilder,
			Count:         1,
			CreationDelay: time.Second * 1,
		}
	}
	f.ExecutePipelineRunTests(t, allTests...)
}

func Test_PipelineRunSingle_Jenkinsfile(t *testing.T) {
	allTests := make([]f.TestPlan, len(JenkinsfileTestBuilders))
	for i, pipelinerunTestBuilder := range JenkinsfileTestBuilders {
		allTests[i] = f.TestPlan{TestBuilder: pipelinerunTestBuilder,
			Count:         1,
			CreationDelay: time.Second * 1,
		}
	}
	f.ExecutePipelineRunTests(t, allTests...)
}

func Test_PipelineRunSingle_Secrets(t *testing.T) {
	allTests := make([]f.TestPlan, len(SecretTestBuilders))
	for i, pipelinerunTestBuilder := range SecretTestBuilders {
		allTests[i] = f.TestPlan{TestBuilder: pipelinerunTestBuilder,
			Count:         1,
			CreationDelay: time.Second * 1,
		}
	}
	f.ExecutePipelineRunTests(t, allTests...)
}
