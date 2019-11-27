package framework

import (
	"reflect"
	"runtime"
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
type PipelineRunTestBuilder = func(string) PipelineRunTest

// TestPlan defines a test plan
type TestPlan struct {
	Name             string
	TestBuilder      PipelineRunTestBuilder
	Parallel         int
	ParallelCreation bool
	CreationDelay    time.Duration
}

func getTestPlanName(plan TestPlan) string {
	name := plan.Name
	if name == "" {
		name = runtime.FuncForPC(reflect.ValueOf(plan.TestBuilder).Pointer()).Name()
	}
	return name
}
