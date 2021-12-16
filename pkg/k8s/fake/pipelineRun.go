package fake

import (
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineRun creates a new pipeline run object.
func PipelineRun(name string, namespace string, spec stewardv1alpha1.PipelineSpec) *stewardv1alpha1.PipelineRun {
	return &stewardv1alpha1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: stewardv1alpha1.SchemeGroupVersion.String(),
			Kind:       "PipelineRun",
		},
		ObjectMeta: ObjectMeta(name, namespace),
		Spec:       spec,
	}
}
