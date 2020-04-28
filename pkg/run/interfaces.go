package run

import (
	"context"

	steward "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativeapis "knative.dev/pkg/apis"
)

// Manager manages runs
type Manager interface {
	Start(ctx context.Context, pipelineRun k8s.PipelineRun) error
	GetRun(ctx context.Context, pipelineRun k8s.PipelineRun) (Run, error)
	Cleanup(ctx context.Context, pipelineRun k8s.PipelineRun) error
}

// Run represents a pipeline run
type Run interface {
	GetStartTime() *metav1.Time
	IsFinished() (bool, steward.Result)
	GetSucceededCondition() *knativeapis.Condition
	GetContainerInfo() *corev1.ContainerState
}
