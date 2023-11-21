package runmgr

import (
	steward "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	runifc "github.com/SAP/stewardci-core/pkg/runctl/run"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	termination "github.com/tektoncd/pipeline/pkg/termination"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativeapis "knative.dev/pkg/apis"
)

const (
	jfrExitCodeErrorContent = 2
	jfrExitCodeErrorConfig  = 3
)

// tektonRun is a runifc.Run based on Tekton.
type tektonRun struct {
	tektonTaskRun *tekton.TaskRun
}

// Compiler check for interface compliance
var _ runifc.Run = (*tektonRun)(nil)

// newRun creates a new tektonRun
func newRun(tektonTaskRun *tekton.TaskRun) *tektonRun {
	return &tektonRun{tektonTaskRun: tektonTaskRun}
}

// GetStartTime implements runifc.Run.
func (r *tektonRun) GetStartTime() *metav1.Time {
	condition := r.getSucceededCondition()
	if condition == nil {
		return nil
	}
	if condition.IsUnknown() && condition.Reason != string(tekton.TaskRunReasonRunning) {
		return nil
	}
	if stepState := r.getJFRStepState(); stepState != nil {
		if stepState.Running != nil {
			return &stepState.Running.StartedAt
		}
		if stepState.Terminated != nil {
			return &stepState.Terminated.StartedAt
		}
	}
	return nil
}

// GetCompletionTime implements runifc.Run.
func (r *tektonRun) GetCompletionTime() *metav1.Time {
	completionTime := r.tektonTaskRun.Status.CompletionTime
	if completionTime != nil {
		return completionTime
	}
	condition := r.getSucceededCondition()
	if condition != nil {
		ltt := condition.LastTransitionTime.Inner
		if !ltt.IsZero() {
			return &ltt
		}
	}

	now := metav1.Now()
	return &now
}

// GetContainerInfo implements runifc.Run.
func (r *tektonRun) GetContainerInfo() *corev1.ContainerState {
	stepState := r.getJFRStepState()
	if stepState == nil {
		return nil
	}
	return &stepState.ContainerState
}

func (r *tektonRun) getSucceededCondition() *knativeapis.Condition {
	return r.tektonTaskRun.Status.GetCondition(knativeapis.ConditionSucceeded)
}

// IsRestartable implements runifc.Run.
func (r *tektonRun) IsRestartable() bool {
	condition := r.getSucceededCondition()
	if condition.IsFalse() {
		// TaskRun finished unsuccessfully, check reason...
		switch condition.Reason {
		case
			string(tekton.TaskRunReasonImagePullFailed),
			"PodCreationFailed":
			return true
		}
	}
	return false
}

// IsFinished implements runifc.Run.
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
	case string(tekton.TaskRunReasonTimedOut):
		return true, steward.ResultTimeout
	case string(tekton.TaskRunReasonFailed):
		jfrStepState := r.getJFRStepState()
		if jfrStepState != nil && jfrStepState.Terminated != nil {
			switch jfrStepState.Terminated.ExitCode {
			case jfrExitCodeErrorContent:
				return true, steward.ResultErrorContent
			case jfrExitCodeErrorConfig:
				return true, steward.ResultErrorConfig
			}
		}
	default:
		// TODO handle other failure reasons like quota exceedance
	}
	return true, steward.ResultErrorInfra
}

// GetMessage implements runifc.Run.
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
		allMessages, err := termination.ParseMessage(zap.S(), msg)
		if err != nil {
			return msg
		}
		for _, singleMessage := range allMessages {
			if singleMessage.Key == jfrResultKey {
				return singleMessage.Value
			}
		}
	}
	return "internal error"
}

// IsDeleted implements runifc.Run.
func (r *tektonRun) IsDeleted() bool {
	return r == nil || r.tektonTaskRun.DeletionTimestamp != nil
}

func (r *tektonRun) getJFRStepState() *tekton.StepState {
	steps := r.tektonTaskRun.Status.Steps
	for _, stepState := range steps {
		if stepState.Name == JFRTaskRunStepName {
			return &stepState
		}
	}
	return nil
}
