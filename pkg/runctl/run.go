package runctl

import (
	steward "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonStatus "github.com/tektoncd/pipeline/pkg/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativeapis "knative.dev/pkg/apis"
)

// Run represents a pipeline run
type Run interface {
	GetStartTime() *metav1.Time
	IsFinished() (bool, steward.Result)
	GetSucceededCondition() *knativeapis.Condition
	GetContainerInfo() *corev1.ContainerState
}

type run struct {
	tektonTaskRun *tekton.TaskRun
}

// NewRun returns new Run
func NewRun(tektonTaskRun *tekton.TaskRun) Run {
	return &run{tektonTaskRun: tektonTaskRun}
}

// GetStartTime returns start time of run if already started
func (r *run) GetStartTime() *metav1.Time {
	return r.tektonTaskRun.Status.StartTime
}

// GetContainerInfo returns the state of the Jenkinsfile Runner container
// as reported in the Tekton TaskRun status.
func (r *run) GetContainerInfo() *corev1.ContainerState {
	stepState := r.getJenkinsfileRunnerStepState()
	if stepState == nil {
		return nil
	}
	return &stepState.ContainerState
}

func (r *run) GetSucceededCondition() *knativeapis.Condition {
	return r.tektonTaskRun.Status.GetCondition(knativeapis.ConditionSucceeded)
}

// IsFinished returns true if run is finished
func (r *run) IsFinished() (bool, steward.Result) {
	condition := r.GetSucceededCondition()
	if condition.IsUnknown() {
		return false, steward.ResultUndefined
	}
	if condition.IsTrue() {
		return true, steward.ResultSuccess
	}
	// TaskRun finished unsuccessfully, check reason...
	switch condition.Reason {
	case tektonStatus.ReasonTimedOut:
		return true, steward.ResultTimeout
	case tektonStatus.ReasonFailed:
		jfrStepState := r.getJenkinsfileRunnerStepState()
		if jfrStepState != nil && jfrStepState.Terminated != nil && jfrStepState.Terminated.ExitCode != 0 {
			return true, steward.ResultErrorContent
		}
	default:
		// TODO handle other failure reasons like quota exceedance
	}
	return true, steward.ResultErrorInfra
}

func (r *run) getJenkinsfileRunnerStepState() *tekton.StepState {
	steps := r.tektonTaskRun.Status.Steps
	if steps != nil {
		for _, stepState := range steps {
			if stepState.Name == tektonClusterTaskJenkinsfileRunnerStep {
				return &stepState
			}
		}
	}
	return nil
}
