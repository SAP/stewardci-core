package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	Progress            TenantCreationProgress `json:"progress"`
	Result              TenantResult           `json:"result"`
	Message             string                 `json:"message"`
	TenantNamespaceName string                 `json:"tenantNamespaceName"`
}

// TenantCreationProgress of the Tenant
type TenantCreationProgress string

const (
	//TenantProgressUndefined Did not start yet
	TenantProgressUndefined TenantCreationProgress = ""
	//TenantProgressInProcess Just started, nothing done yet
	TenantProgressInProcess TenantCreationProgress = "InProcess"
	//TenantProgressCreateNamespace current step create namespace
	TenantProgressCreateNamespace TenantCreationProgress = "CreateNamespace"
	//TenantProgressGetServiceAccount current step get service account
	TenantProgressGetServiceAccount TenantCreationProgress = "GetServiceAccount"
	//TenantProgressAddRoleBinding current step add role binding
	TenantProgressAddRoleBinding TenantCreationProgress = "AddRoleBinding"
	//TenantProgressFinalize current step finalize, all steps before were successful.
	TenantProgressFinalize TenantCreationProgress = "Finalize"
	//TenantProgressFinished process finished
	TenantProgressFinished TenantCreationProgress = "Finished"
)

// TenantResult of the tenant processing
type TenantResult string

const (
	// TenantResultUndefined - undefined TenantResult
	TenantResultUndefined TenantResult = ""
	// TenantResultSuccess - the tenant has been set up successfully
	TenantResultSuccess TenantResult = "success"
	// TenantResultErrorInfra - the tentant setup failed due to an infrastructure problem
	TenantResultErrorInfra TenantResult = "error_infra"
	// TenantResultErrorContent -  the tenant setup failed due to an content problem
	TenantResultErrorContent TenantResult = "error_content"
)
