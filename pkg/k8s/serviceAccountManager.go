package k8s

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	checkRoleExistence = true
)

//ServiceAccountManager manages serviceAccounts
type ServiceAccountManager interface {
	CreateServiceAccount(name string, scmCloneSecretName string, pullSecretName []string) (*ServiceAccountWrap, error)
	GetServiceAccount(name string) (*ServiceAccountWrap, error)
}

type serviceAccountManager struct {
	factory ClientFactory
	client  corev1.ServiceAccountInterface
}

// ServiceAccountWrap wraps a Service Account and enriches it with futher things
type ServiceAccountWrap struct {
	factory ClientFactory
	cache   *v1.ServiceAccount
}

// RoleName to be attached
type RoleName string

//NewServiceAccountManager creates ServiceAccountManager
func NewServiceAccountManager(factory ClientFactory, namespace string) ServiceAccountManager {
	return &serviceAccountManager{
		factory: factory,
		client:  factory.CoreV1().ServiceAccounts(namespace),
	}
}

// CreateServiceAccount creates a service account on the cluster
//   name					name of the service account
//   scmCloneSecretName		(optional) the scm clone secret to attach to this service account (e.g. for fetching the Jenkinsfile)
//   pullSecretNames		(optional) a lsit of pull secrets to attach to this service account (e.g. for pulling the Jenkinsfile Runner image)
func (c *serviceAccountManager) CreateServiceAccount(name string, scmCloneSecretName string, pullSecretNames []string) (*ServiceAccountWrap, error) {
	serviceAccount := &v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if scmCloneSecretName != "" {
		secretList := make([]v1.ObjectReference, 1)
		secretList[0] = v1.ObjectReference{Name: scmCloneSecretName}
		serviceAccount.Secrets = secretList
	}
	for index, pullSecretName := range pullSecretNames {
		refList := make([]v1.LocalObjectReference, len(pullSecretNames))
		refList[index] = v1.LocalObjectReference{Name: pullSecretName}
		serviceAccount.ImagePullSecrets = refList
	}

	account, err := c.client.Create(serviceAccount)
	return &ServiceAccountWrap{
		factory: c.factory,
		cache:   account,
	}, err
}

// GetServiceAccount gets a ServiceAccount from the cluster
func (c *serviceAccountManager) GetServiceAccount(name string) (serviceAccount *ServiceAccountWrap, err error) {
	var account *v1.ServiceAccount
	if account, err = c.client.Get(name, metav1.GetOptions{}); err != nil {
		return
	}
	serviceAccount = &ServiceAccountWrap{
		factory: c.factory,
		cache:   account,
	}
	return
}

// AddRoleBinding creates a role binding in the targetNamespace connecting the service account with the specified cluster role
func (a *ServiceAccountWrap) AddRoleBinding(clusterRole RoleName, targetNamespace string) (*v1beta1.RoleBinding, error) {

	//Check if cluster role exists
	if checkRoleExistence {
		clusterRole, err := a.factory.RbacV1beta1().ClusterRoles().Get(string(clusterRole), metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if clusterRole == nil {
			return nil, fmt.Errorf("Cluster Role '%v' does not exist", clusterRole)
		}
	}

	//Create role binding
	roleBindingClient := a.factory.RbacV1beta1().RoleBindings(targetNamespace)
	subjects := make([]v1beta1.Subject, 1)
	subjects[0] = v1beta1.Subject{Kind: "ServiceAccount", Name: a.cache.GetName(), Namespace: a.cache.GetNamespace()}
	roleBinding := &v1beta1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: string(clusterRole), Namespace: targetNamespace},
		Subjects: subjects,
		RoleRef:  v1beta1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: string(clusterRole)}}

	return roleBindingClient.Create(roleBinding)
}

// GetServiceAccount returns *v1.ServiceAccount
func (a *ServiceAccountWrap) GetServiceAccount() *v1.ServiceAccount {
	return a.cache
}
