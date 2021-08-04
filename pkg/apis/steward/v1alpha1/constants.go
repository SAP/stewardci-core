package v1alpha1

import (
	"github.com/SAP/stewardci-core/pkg/apis/steward"
)

// annotations
const (
	// AnnotationTenantNamespacePrefix is the key of the annotation
	// of a Steward client namespace defining the prefix of tenant namespaces
	// belonging to this client.
	AnnotationTenantNamespacePrefix = steward.GroupName + "/tenant-namespace-prefix"

	// AnnotationTenantNamespaceSuffixLength is the key of the annotation
	// of a Steward client namespace defining the number of characters used for
	// the random suffix of a tenant namespace name.
	AnnotationTenantNamespaceSuffixLength = steward.GroupName + "/tenant-namespace-suffix-length"

	// AnnotationTenantRole is the key of the annotation of a Steward client
	// namespace defining the name of the ClusterRole to be assigned to the
	// default service account of a tenant namespace.
	AnnotationTenantRole = steward.GroupName + "/tenant-role"

	// AnnotationSecretRename is the key of the annotation used to rename a secret.
	// If this annotation is set on a secret it will be created in the run namespace
	// with this name if it is listed in the pipelineRuns spec.secrets list.
	AnnotationSecretRename = steward.GroupName + "/secret-rename-to"
)

// labels
const (
	// LabelSystemManaged is the key of the label whose presence indicates
	// that this resource is managed by the Steward system and should not be
	// modified otherwise.
	// The value of the label is ignored and should be empty.
	LabelSystemManaged = steward.GroupName + "/system-managed"

	// LabelOwnerClientName is the key of the label that identifies the Steward
	// _client_ that the labelled object is owned by.
	// As Steward clients are currently represented by K8s namespaces only,
	// the label value is the name of the respective client namespace.
	// This may change in the future when Steward clients are represented by
	// dedicated custom resources.
	LabelOwnerClientName = steward.GroupName + "/owner-client-name"

	// LabelOwnerClientNamespace is the key of the label that identifies the
	// namespace assigned to the Steward _client_ that the labelled object is
	// owned by.
	LabelOwnerClientNamespace = steward.GroupName + "/owner-client-namespace"

	// LabelOwnerTenantName is the key of the label that identifies the Steward
	// _tenant_ that the labelled object is owned by.
	// The label value is the name of the Tenant custom resource.
	LabelOwnerTenantName = steward.GroupName + "/owner-tenant-name"

	// LabelOwnerTenantNamespace is the key of the label that identifies the
	// namespace assigned to the Steward _tenant_ that the labelled object is
	// owned by.
	LabelOwnerTenantNamespace = steward.GroupName + "/owner-tenant-namespace"

	// LabelOwnerPipelineRunName is the key of the label that identifies the
	// Steward _pipeline run_ that the labelled object is owned by.
	// The label value is the name of the PipelineRun custom resource.
	LabelOwnerPipelineRunName = steward.GroupName + "/owner-pipelinerun-name"
)

// K8s events
const (
	// EventReasonPreparingFailed is the reason for a event occuring when the run controller
	// faces an intermittent error during preparing phase.
	EventReasonPreparingFailed = "PreparingFailed"

	// EventReasonWaitingFailed is the reason for a event occuring when the run controller
	// faces an intermittent error during wait phase.
	EventReasonWaitingFailed = "WaitingFailed"

	// EventReasonRunningFailed is the reason for a event occuring when the run controller
	// faces an intermittent error during running phase.
	EventReasonRunningFailed = "RunningFailed"

	// EventReasonCleaningFailed is the reason for a event occuring when the run controller
	// faces an intermittent error during cleanup phase.
	EventReasonCleaningFailed = "CleaningFailed"

	// EventReasonLoadPipelineRunsConfigFailed is the reason for an event occuring when the
	// loading of the pipeline runs configuration fails.
	EventReasonLoadPipelineRunsConfigFailed = "LoadPipelineRunsConfigFailed"

	// EventReasonMaintenanceMode is the reason for an event occuring when a pipeline
	// run is not started due to maintenance mode
	EventReasonMaintenanceMode = "MaintenanceMode"

	// MaintenanceModeConfigMapName is the name of the config map to enable the maintenance mode
	MaintenanceModeConfigMapName = "steward-maintenance-mode"

	// MaintenanceModeKeyName is the name of the key to enable the maintenance mode
	MaintenanceModeKeyName = "maintenanceMode"
)
