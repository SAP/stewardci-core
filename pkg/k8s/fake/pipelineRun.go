package fake

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineRun creates a new pipeline run object.
func PipelineRun(name string, namespace string, spec api.PipelineSpec) *api.PipelineRun {
	typeMeta := metav1.TypeMeta{Kind: "PipelineRun", APIVersion: "steward.sap.com/v1alpha1"}
	objectMeta := ObjectMeta(name, namespace)
	return &api.PipelineRun{
		TypeMeta:   typeMeta,
		ObjectMeta: objectMeta,
		Spec:       spec,
	}
}
