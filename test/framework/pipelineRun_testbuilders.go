package framework

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

// PipelineRunTest is a test for a pipeline run
type PipelineRunTest struct {
	PipelineRun *api.PipelineRun
	Secrets     []*v1.Secret
	Check       PipelineRunCheck
	Expected    string
	Timeout     time.Duration
}

// PipelineRunTestBuilder is a funciton creating a PipelineRunTest for a defined Namespace
// and a buildID for the elasticsearch
type PipelineRunTestBuilder = func(string, *api.CustomJSON) PipelineRunTest

// TestPlan defines a test plan
type TestPlan struct {
	Name             string
	TestBuilder      PipelineRunTestBuilder
	Count            int
	ParallelCreation bool
	CreationDelay    time.Duration
	Cleanup          bool
}

func getTestPlanName(plan TestPlan) string {
	name := plan.Name
	if name == "" {
		name = runtime.FuncForPC(reflect.ValueOf(plan.TestBuilder).Pointer()).Name()
		names := strings.Split(name, "/")

		name = names[len(names)-1]
		names = strings.Split(name, ".")
		name = names[1]
	}
	if plan.Count > 1 {
		delay := "parallel"
		if !plan.ParallelCreation {
			if plan.CreationDelay > 0 {
				delay = fmt.Sprintf("delay:%.1fs", plan.CreationDelay.Seconds())
			} else {
				delay = "nodelay"
			}
		}
		return fmt.Sprintf("%s_c%d_%s", name, plan.Count, delay)
	}
	return name
}
