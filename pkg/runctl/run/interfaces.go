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
	// CreateEnv creates a new isolated environment for a new run.
	// If an environment exists already, it will be removed first.
	CreateEnv(ctx context.Context, pipelineRun k8s.PipelineRun, pipelineRunsConfig *cfg.PipelineRunsConfigStruct) (string, string, error)

	// CreateRun creates a new run in the prepared environment.
	// Especially fails if the environment does not exist or a run exists
	// already.
	CreateRun(ctx context.Context, pipelineRun k8s.PipelineRun, pipelineRunsConfig *cfg.PipelineRunsConfigStruct) error

	// GetRun returns the run or nil if a run has not been created yet.
	GetRun(ctx context.Context, pipelineRun k8s.PipelineRun) (Run, error)

	// DeleteRun deletes a task run for a given pipeline run.
	DeleteRun(ctx context.Context, pipelineRun k8s.PipelineRun) error

	// DeleteEnv removes an existing environment.
	// If no environment exists, it succeeds.
	DeleteEnv(ctx context.Context, pipelineRun k8s.PipelineRun) error
}

// Run represents a pipeline run
type Run interface {
	// GetStartTime returns the timestamp when the run actually started.
	// Initialization steps should be excluded as far as possible.
	// Returns nil if the run has not been started yet.
	GetStartTime() *metav1.Time

	// GetCompletionTime returns the timestamp of the run's completion.
	// Teardown steps should be excluded as far as possible.
	// Returns nil if the run has never been started or has not completed yet.
	GetCompletionTime() *metav1.Time

	// IsFinished returns true if the run is finished.
	// Note that a run can be finished without having been started, i.e.
	// there was an error.
	IsFinished() (bool, steward.Result)

	// IsRestartable returns true if run finished unsuccessfully and can be
	// restarted with a possibly successful result.
	IsRestartable() bool

	// GetContainerInfo returns the state of the Jenkinsfile Runner container
	// as reported in the Tekton TaskRun status.
	GetContainerInfo() *corev1.ContainerState

	// GetMessage returns the status message.
	GetMessage() string

	// IsDeleted returns true if the receiver is nil or is marked as deleted.
	IsDeleted() bool
}

// SecretManager manages secrets of a pipelinerun
type SecretManager interface {
	// CopyAll copies all the required secrets of a pipeline run to the
	// respective run namespace.
	CopyAll(ctx context.Context, pipelineRun k8s.PipelineRun) (string, []string, error)
}
