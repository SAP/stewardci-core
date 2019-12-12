package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineRun is a Kubernetes custom resource type representing the execution
// of a pipeline.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PipelineRun struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PipelineSpec `json:"spec"`

	// +optional
	Status PipelineStatus `json:"status"`
}

// PipelineRunList is a list of PipelineRun objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PipelineRunList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineRun `json:"items"`
}

// PipelineSpec is the spec of a PipelineRun
type PipelineSpec struct {

	// JenkinsFile contains the configuration of the Jenkins pipeline definition
	// to be executed.
	JenkinsFile JenkinsFile `json:"jenkinsFile"`

	// Args contains the key-value parameters to pass to the pipeline.
	// +optional
	Args map[string]string `json:"args"`

	// Secrets is the list of secrets to be made available to the pipeline
	// execution. Each entry in the list is the name of a Kubernetes `v1/Secret`
	// resource object in the same namespace as the PipelineRun object itself.
	// +optional
	Secrets []string `json:"secrets"`

	// ImagePullSecrets is the list of image pull secrets required by the
	// pipeline run to pull images of custom containers from private registries.
	// Each entry in the list is the name of a Kubernetes `v1/Secret` resource
	// object of type `kubernetes.io/dockerconfigjson` in the same namespace as
	// the PipelineRun object itself.
	// +optional
	ImagePullSecrets []string `json:"imagePullSecrets"`

	// Intent is the intention of the client regarding the way this pipeline run
	// should be processed. The value `run` indicates that the pipeline should
	// run to completion, while the value `abort` indicates that the pipeline
	// processing should be stopped as soon as possible. An empty string value
	// is equivalent to value `run`.
	// TODO: Controller should set intent=run explicitely if not set
	// +optional
	Intent Intent `json:"intent"`

	// Logging contains the logging configuration.
	// +optional
	Logging *Logging `json:"logging"`

	// RunDetails provides metadata for a pipeline run which is evaluated by the
	// Jenkinsfile Runner.
	// +optional
	RunDetails *PipelineRunDetails `json:"runDetails"`
}

// JenkinsFile represents the location from where to get the pipeline
type JenkinsFile struct {

	// URL is the URL of the Git repository containing the pipeline definition
	// (aka `Jenkinsfile`).
	URL string `json:"repoUrl"`

	// Revision is the revision of the pipeline Git repository to be used, e.g.
	// `master`.
	Revision string `json:"revision"`

	// Path is the relative pathname of the pipeline definition file in the
	// repository check-out, typically `Jenkinsfile`.
	Path string `json:"relativePath"`

	// RepoAuthSecret is the name of the Kubernetes `v1/Secret` resource object
	// of type `kubernetes.io/basic-auth` that contains the username and
	// password for authentication when cloning from `spec.jenkinsFile.repoUrl`.
	// +optional
	RepoAuthSecret string `json:"repoAuthSecret"`
}

// Logging contains all logging-specific configuration.
type Logging struct {

	// Elasticsearch is the configuration for pipeline logging to Elasticsearch.
	// If not specified, logging to Elasticsearch is disabled and the default
	// Jenkins log implementation is used (stdout of Jenkinsfile Runner
	// container).
	// +optional
	Elasticsearch *Elasticsearch `json:"elasticsearch"`
}

// Elasticsearch contains logging configuration for the
// Elasticsearch log implementation.
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
	FinishedAt *metav1.Time `json:"finishedAt,omitempty"`

	State        State                 `json:"state"`
	StateDetails StateItem             `json:"stateDetails"`
	StateHistory []StateItem           `json:"stateHistory"`
	Result       Result                `json:"result"`
	Container    corev1.ContainerState `json:"container,omitempty"`
	MessageShort string                `json:"messageShort"`
	Message      string                `json:"message"`
	History      []string              `json:"history"`
	Namespace    string                `json:"namespace"`
}

// StateItem holds start and end time of a state in the history
type StateItem struct {
	State      State       `json:"state"`
	StartedAt  metav1.Time `json:"startedAt"`
	FinishedAt metav1.Time `json:"finishedAt,omitempty"`
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
	// ResultAborted - the pipeline run has been aborted
	ResultAborted Result = "aborted"
	// ResultTimeout - the pipeline run timed out
	ResultTimeout Result = "timeout"
)

// Intent denotes how the pipeline run should be handled
type Intent string

const (
	// IntentRun indicates that the pipeline should run to completion.
	IntentRun Intent = "run"
	// IntentAbort indicates that the pipeline run should be aborted
	// if it is not completed already.
	IntentAbort Intent = "abort"
)

// PipelineRunDetails provides metadata for a pipeline run which is evaluated by
// the Jenkinsfile Runner.
type PipelineRunDetails struct {

	// JobName is the name of the job this pipeline run belongs to. It is used
	// as the name of the Jenkins job and therefore must be a valid Jenkins job
	// name. If empty, a default name will be used for the Jenkins job.
	// TODO: Regex in CRD to validate "valid Jenkins job names"
	// +optional
	JobName string `json:"jobName,omitempty"`

	// SequenceNumber is the sequence number of the pipeline run, which
	// translates into the build number of the Jenkins job.
	// +optional
	SequenceNumber int32 `json:"sequenceNumber,omitempty"`

	// Cause is a textual description of the cause of this pipeline run. Will be
	// set as cause of the Jenkins job. If empty, no cause information
	// will be available.
	// +optional
	Cause string `json:"cause,omitempty"`
}
