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

	// LabelSystemManaged is the key of the label whose presence indicates
	// that this resource is managed by the Steward system and should not be
	// modified otherwise.
	// The value of the label is ignored and should be empty.
	LabelSystemManaged = steward.GroupName + "/system-managed"
)
