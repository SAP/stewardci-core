package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineRun is a K8s custom resource representing a singe pipeline run
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PipelineRun struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Status PipelineStatus `json:"status"`
	Spec   PipelineSpec   `json:"spec"`
}

// PipelineRunList is a list of PipelineRun objects
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PipelineRunList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineRun `json:"items"`
}

// PipelineSpec is the spec of a PipelineRun
type PipelineSpec struct {
	JenkinsFile JenkinsFile       `json:"jenkinsFile"`
	Args        map[string]string `json:"args"`
	// +optional
	Secrets []string `json:"secrets"`
	// +optional
	ImagePullSecrets []string `json:"imagePullSecrets"`
	Intent           Intent   `json:"intent"`
	Logging          *Logging `json:"logging"`
	// +optional
	RunDetails *PipelineRunDetails `json:"runDetails"`
}

// JenkinsFile represents the location from where to get the pipeline
type JenkinsFile struct {
	URL      string `json:"repoUrl"`
	Revision string `json:"revision"`
	Path     string `json:"relativePath"`
	// +optional
	RepoAuthSecret string `json:"repoAuthSecret"`
}

// Logging contains all logging-specific configuration.
type Logging struct {
	Elasticsearch *Elasticsearch `json:"elasticsearch"`
}

// Elasticsearch contains logging configuration for the
// Elasticsearch log implementation
type Elasticsearch struct {
	// The identifier of this pipeline run, attached as
	// field `runid` to each log entry.
	// It can by any JSON value (object, array, string,
	// number, bool).
	RunID *CustomJSON `json:"runID"`
}

// PipelineStatus represents the status of the pipeline
type PipelineStatus struct {
	// StartedAt is the time the pipeline run has been started.
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// FinishedAt is the time the pipeline run has been finished.
	// +optional
	FinishedAt   *metav1.Time          `json:"finishedAt,omitempty"`
	State        State                 `json:"state"`
	StateDetails StateItem             `json:"stateDetails"`
	StateHistory []StateItem           `json:"stateHistory"`
	Result       Result                `json:"result"`
	Container    corev1.ContainerState `json:"container,omitempty"`
	LogURL       string                `json:"logUrl"`
	MessageShort string                `json:"messageShort"`
	Message      string                `json:"message"`
	History      []string              `json:"history"`
	Namespace    string                `json:"namespace"`
}

// StateItem holds start and end time of a state in the history
type StateItem struct {
	State      State       `json:"state"`
	StartedAt  metav1.Time `json:"startedAt"`
	FinishedAt metav1.Time `json:"finishedAt"`
}

// State represents the state
type State string

const (
	// StateUndefined - the state was not yet set
	StateUndefined State = ""
	// StateNew - pipeline run is first checked by the controller
	StateNew State = "new"
	// StatePreparing - the namespace for the execution is prepared
	StatePreparing State = "preparing"
	// StateWaiting - the pipeline run is waiting to be processed
	StateWaiting State = "waiting"
	// StateRunning - the pipeline is running
	StateRunning State = "running"
	// StateCleaning - cleanup is ongoing
	StateCleaning State = "cleaning"
	// StateFinished - the pipeline run has finished
	StateFinished State = "finished"
)

// Result of the pipeline run
type Result string

const (
	// ResultUndefined - undefined result
	ResultUndefined Result = ""
	// ResultSuccess - the pipeline run was processed successfully
	ResultSuccess Result = "success"
	// ResultErrorInfra - the pipeline run failed due to an infrastructure problem
	ResultErrorInfra Result = "error_infra"
	// ResultErrorContent -  the pipeline run failed due to an content problem
	ResultErrorContent Result = "error_content"
	// ResultKilled - the pipeline run has been cancelled
	ResultKilled Result = "killed"
	// ResultTimeout - the pipeline run timed out
	ResultTimeout Result = "timeout"
)

// Intent denotes how the pipeline run should be handled
type Intent string

const (
	// IntentRun - run the pipeline
	IntentRun Intent = "run"
	// IntentKill - cancel the pipeline run (if still running)
	IntentKill Intent = "kill"
)

// PipelineRunDetails provies details of a pipeline run which are passed to the jenkinsfile-runner.
type PipelineRunDetails struct {
	// JobName is the name of the job which is instantiated by the run.
	JobName string `json:"jobName"`
	// SequenceNumber is a sequential number of the run
	SequenceNumber int `json:"sequenceNumber"`
	// Cause is the cause which triggered the run, e.g. a SCM change, an user action or a timer.
	Cause string `json:"cause"`
}
