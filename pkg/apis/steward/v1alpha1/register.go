package v1alpha1

import (
	x "github.com/SAP/stewardci-core/pkg/apis/steward"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersion is the version for the scheme
const GroupVersion = "v1alpha1"

// SchemeGroupVersion ...
var SchemeGroupVersion = schema.GroupVersion{Group: x.GroupName, Version: GroupVersion}

var (
	// SchemeBuilder builds the scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme ...
	AddToScheme = SchemeBuilder.AddToScheme
)

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&PipelineRun{},
		&PipelineRunList{},
		&Tenant{},
		&TenantList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
