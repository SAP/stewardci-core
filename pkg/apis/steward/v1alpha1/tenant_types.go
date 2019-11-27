package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativeapis "knative.dev/pkg/apis"
	knativeduck "knative.dev/pkg/apis/duck/v1"
)

// Tenant is representing a Tenant and its status
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Tenant struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Status TenantStatus `json:"status"`
	Spec   TenantSpec   `json:"spec"`
}

// TenantList is a list of Tenants
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

// TenantSpec is a spec of a Tenant
type TenantSpec struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// TenantStatus contains the status of a Tenant
type TenantStatus struct {
	knativeduck.Status `json:",inline"`

	TenantNamespaceName string `json:"tenantNamespaceName,omitempty"`
}

var tenantConditionSet = knativeapis.NewLivingConditionSet()

// GetCondition returns the condition matching the given condition type.
func (s *TenantStatus) GetCondition(condType knativeapis.ConditionType) *knativeapis.Condition {
	return tenantConditionSet.Manage(s).GetCondition(condType)
}

// SetCondition sets the given condition.
func (s *TenantStatus) SetCondition(cond *knativeapis.Condition) {
	if cond != nil {
		tenantConditionSet.Manage(s).SetCondition(*cond)
	}
}
