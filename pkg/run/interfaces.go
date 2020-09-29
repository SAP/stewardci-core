package run

import (
	steward "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Manager manages runs
type Manager interface {
	Start(pipelineRun k8s.PipelineRun) error
	GetRun(pipelineRun k8s.PipelineRun) (Run, error)
	Cleanup(pipelineRun k8s.PipelineRun) error
}

// Run represents a pipeline run
type Run interface {
	GetStartTime() *metav1.Time
	IsFinished() (bool, steward.Result)
	GetContainerInfo() *corev1.ContainerState
	GetMessage() string
}
