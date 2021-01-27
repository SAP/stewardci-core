package v1alpha1

import (
	"github.com/SAP/stewardci-core/pkg/apis/steward"
)

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

	// LabelSystemManaged is the key of the label whose presence indicates
	// that this resource is managed by the Steward system and should not be
	// modified otherwise.
	// The value of the label is ignored and should be empty.
	LabelSystemManaged = steward.GroupName + "/system-managed"

	// EventReasonPreparingFailed is the reason for a event occuring when the run controller
	// faces an intermittent error during preparing phase.
	EventReasonPreparingFailed = "PreparingFailed"

	// EventReasonWaitingFailed is the reason for a event occuring when the run controller
	// faces an intermittent error during wait phase.
	EventReasonWaitingFailed = "WaitingFailed"

	// EventReasonRunningFailed is the reason for a event occuring when the run controller
	// faces an intermittent error during running phase.
	EventReasonRunningFailed = "RunningFailed"

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
