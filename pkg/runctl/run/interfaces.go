package run

import (
	"context"

	steward "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	"github.com/SAP/stewardci-core/pkg/runctl/cfg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Manager manages runs
type Manager interface {
	Prepare(ctx context.Context, pipelineRun k8s.PipelineRun, pipelineRunsConfig *cfg.PipelineRunsConfigStruct) (string, string, error)
	Start(ctx context.Context, pipelineRun k8s.PipelineRun, pipelineRunsConfig *cfg.PipelineRunsConfigStruct) error
	GetRun(ctx context.Context, pipelineRun k8s.PipelineRun) (Run, error)
	Cleanup(ctx context.Context, pipelineRun k8s.PipelineRun) error
	DeleteRun(ctx context.Context, pipelineRun k8s.PipelineRun) error
}

// Run represents a pipeline run
type Run interface {
	GetStartTime() *metav1.Time
	IsRestartable() bool
	IsFinished() (bool, steward.Result)
	GetCompletionTime() *metav1.Time
	GetContainerInfo() *corev1.ContainerState
	GetMessage() string
}

// SecretManager manages secrets of a pipelinerun
type SecretManager interface {
	CopyAll(ctx context.Context, pipelineRun k8s.PipelineRun) (string, []string, error)
}
