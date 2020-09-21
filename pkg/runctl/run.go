package runctl

import (
	steward "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	run "github.com/SAP/stewardci-core/pkg/run"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	termination "github.com/tektoncd/pipeline/pkg/termination"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativeapis "knative.dev/pkg/apis"
)

type tektonRun struct {
	tektonTaskRun *tekton.TaskRun
}

// NewRun returns new Run
func NewRun(tektonTaskRun *tekton.TaskRun) run.Run {
	return &tektonRun{tektonTaskRun: tektonTaskRun}
}

// GetStartTime returns start time of run if already started
func (r *tektonRun) GetStartTime() *metav1.Time {
	return r.tektonTaskRun.Status.StartTime
}

// GetContainerInfo returns the state of the Jenkinsfile Runner container
// as reported in the Tekton TaskRun status.
func (r *tektonRun) GetContainerInfo() *corev1.ContainerState {
	stepState := r.getJenkinsfileRunnerStepState()
	if stepState == nil {
		return nil
	}
	return &stepState.ContainerState
}

func (r *tektonRun) getSucceededCondition() *knativeapis.Condition {
	return r.tektonTaskRun.Status.GetCondition(knativeapis.ConditionSucceeded)
}

// IsFinished returns true if run is finished
func (r *tektonRun) IsFinished() (bool, steward.Result) {
	condition := r.getSucceededCondition()
	if condition.IsUnknown() {
		return false, steward.ResultUndefined
	}
	if condition.IsTrue() {
		return true, steward.ResultSuccess
	}
	// TaskRun finished unsuccessfully, check reason...
	switch condition.Reason {
	case tekton.TaskRunReasonTimedOut.String():
		return true, steward.ResultTimeout
	case tekton.TaskRunReasonFailed.String():
		jfrStepState := r.getJenkinsfileRunnerStepState()
		if jfrStepState != nil && jfrStepState.Terminated != nil && jfrStepState.Terminated.ExitCode != 0 {
			return true, steward.ResultErrorContent
		}
	default:
		// TODO handle other failure reasons like quota exceedance
	}
	return true, steward.ResultErrorInfra
}

// GetMessage returns the termination message
func (r *tektonRun) GetMessage() string {
	var msg string

	containerInfo := r.GetContainerInfo()
	if containerInfo != nil && containerInfo.Terminated != nil {
		msg = containerInfo.Terminated.Message
	}
	if msg == "" {
		cond := r.getSucceededCondition()
		if cond != nil {
			return cond.Message
		}
	} else {
		allMessages, err := termination.ParseMessage(msg)
		if err == nil {
			for _, singleMessage := range allMessages {
				if singleMessage.Key == jfrResultKey {
					return singleMessage.Value
				}
			}
		} else {
			return msg
		}
	}
	return "internal error"
}

func (r *tektonRun) getJenkinsfileRunnerStepState() *tekton.StepState {
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
