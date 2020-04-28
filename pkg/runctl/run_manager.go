package runctl

import (
	"context"

	"github.com/SAP/stewardci-core/pkg/k8s"
	runi "github.com/SAP/stewardci-core/pkg/run"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type runManager struct {
	pipelineRunsConfig *pipelineRunsConfigStruct
}

func EnsureRunManager(ctx context.Context, pipelineRunsConfig *pipelineRunsConfigStruct) context.Context {
	result := runi.GetRunManager(ctx)
	if result == nil {
		rm := &runManager{
			pipelineRunsConfig: pipelineRunsConfig}

		return runi.WithRunManager(ctx, rm)
	} else {
		return ctx
	}
}

// Start prepares the isolated environment for a new run and starts
// the run in this environment.
func (r *runManager) Start(ctx context.Context, pipelineRun k8s.PipelineRun) error {
	var err error
	instance := &runInstance{
		pipelineRun:        pipelineRun,
		pipelineRunsConfig: *r.pipelineRunsConfig,
	}

	err = instance.prepareRunNamespace(ctx)
	if err != nil {
		return err
	}
	err = instance.createTektonTaskRun(ctx)
	if err != nil {
		return err
	}

	return nil
}

// Get based on a pipelineRun
func (_ *runManager) GetRun(ctx context.Context, pipelineRun k8s.PipelineRun) (runi.Run, error) {
	namespace := pipelineRun.GetRunNamespace()
	run, err := k8s.GetClientFactory(ctx).TektonV1alpha1().TaskRuns(namespace).Get(tektonTaskRunName, metav1.GetOptions{})
	return NewRun(run), err
}

// Cleanup a run based on a pipelineRun
func (_ *runManager) Cleanup(ctx context.Context, pipelineRun k8s.PipelineRun) error {
	instance := &runInstance{
		pipelineRun:  pipelineRun,
		runNamespace: pipelineRun.GetRunNamespace(),
	}
	return instance.cleanup(ctx)
}
