package stewardlabels

import (
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LabelAsSystemManaged sets label `steward.sap.com/system-managed` at
// the given object.
func LabelAsSystemManaged(obj metav1.Object) {
	if obj == nil {
		return
	}
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[stewardv1alpha1.LabelSystemManaged] = ""
	obj.SetLabels(labels)
}

// LabelAsIgnore sets label `steward.sap.com/ignore` at
// the given object.
func LabelAsIgnore(obj metav1.Object) {
	if obj == nil {
		return
	}
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[stewardv1alpha1.LabelIgnore] = ""
	obj.SetLabels(labels)
}

// IsLabelledAsIgnore return whether the given resource object is labelled
// as to be ignored by Steward.
func IsLabelledAsIgnore(obj metav1.Object) bool {
	_, exists := obj.GetLabels()[stewardv1alpha1.LabelIgnore]
	return exists
}

// LabelAsOwnedByPipelineRun sets some labels on `obj` that identify it
// as owned by the given Steward pipeline run.
// Fails if there's a conflict with existing labels, e.g. `obj` is labelled
// as owned by another Steward pipeline run.
func LabelAsOwnedByPipelineRun(obj metav1.Object, owner *stewardv1alpha1.PipelineRun) error {
	if obj == nil {
		return nil
	}
	return propagate(obj, owner, map[string]string{
		stewardv1alpha1.LabelOwnerPipelineRunName: owner.GetName(),
	})
}
