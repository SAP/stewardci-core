package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	checkRoleExistence = true
)

//ServiceAccountManager manages serviceAccounts
type ServiceAccountManager interface {
	CreateServiceAccount(ctx context.Context, name string, pipelineCloneSecretName string, imagePullSecretNames []string) (*ServiceAccountWrap, error)
	GetServiceAccount(ctx context.Context, name string) (*ServiceAccountWrap, error)
}

type serviceAccountManager struct {
	factory ClientFactory
	client  corev1client.ServiceAccountInterface
}

// ServiceAccountWrap wraps a Service Account and enriches it with futher things
type ServiceAccountWrap struct {
	factory ClientFactory
	cache   *corev1.ServiceAccount
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
//   pipelineCloneSecretName	(optional) the name of the secret to be used to authenticate at the Git repository hosting the pipeline definition.
//   imagePullSecretNames		(optional) a list of image pull secrets to attach to this service account (e.g. for pulling the Jenkinsfile Runner image)
func (c *serviceAccountManager) CreateServiceAccount(ctx context.Context, name string, pipelineCloneSecretName string, imagePullSecretNames []string) (*ServiceAccountWrap, error) {
	serviceAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name}}
	serviceAccountWrap := &ServiceAccountWrap{
		factory: c.factory,
		cache:   serviceAccount,
	}

	if pipelineCloneSecretName != "" {
		serviceAccountWrap.AttachSecrets(pipelineCloneSecretName)
	}
	serviceAccountWrap.AttachImagePullSecrets(imagePullSecretNames...)

	serviceAccount, err := c.client.Create(ctx, serviceAccount, metav1.CreateOptions{})
	serviceAccountWrap.cache = serviceAccount
	return serviceAccountWrap, err
}

// GetServiceAccount gets a ServiceAccount from the cluster
func (c *serviceAccountManager) GetServiceAccount(ctx context.Context, name string) (serviceAccount *ServiceAccountWrap, err error) {
	var account *corev1.ServiceAccount
	if account, err = c.client.Get(ctx, name, metav1.GetOptions{}); err != nil {
		return
	}
	serviceAccount = &ServiceAccountWrap{
		factory: c.factory,
		cache:   account,
	}
	return
}

// AttachSecrets attaches a number of secrets to the service account.
// It does NOT create or update the resource via the underlying client.
func (a *ServiceAccountWrap) AttachSecrets(secretNames ...string) {
	if len(secretNames) == 0 {
		return
	}

	secretRefs := a.cache.Secrets

	haveSecretAlready := func(name string) bool {
		for _, secretRef := range secretRefs {
			if secretRef.Name == name {
				return true
			}
		}
		return false
	}

	changed := false
	for _, secretName := range secretNames {
		if secretName == "" {
			continue
		}
		if !haveSecretAlready(secretName) {
			secretRef := corev1.ObjectReference{Name: secretName}
			secretRefs = append(secretRefs, secretRef)
			changed = true
		}
	}

	if changed {
		a.cache.Secrets = secretRefs
	}
}

// AttachImagePullSecrets attaches a number of secrets to the service account.
// It does NOT create or update the resource via the underlying client.
func (a *ServiceAccountWrap) AttachImagePullSecrets(secretNames ...string) {
	if len(secretNames) == 0 {
		return
	}

	secretRefs := a.cache.ImagePullSecrets

	haveSecretAlready := func(name string) bool {
		for _, secretRef := range secretRefs {
			if secretRef.Name == name {
				return true
			}
		}
		return false
	}

	changed := false
	for _, secretName := range secretNames {
		if secretName == "" {
			continue
		}
		if !haveSecretAlready(secretName) {
			secretRef := corev1.LocalObjectReference{Name: secretName}
			secretRefs = append(secretRefs, secretRef)
			changed = true
		}
	}

	if changed {
		a.cache.ImagePullSecrets = secretRefs
	}
}

// SetDoAutomountServiceAccountToken sets the `automountServiceAccountToken` flag in the
// service account spec.
// It does NOT create or update the resource via the underlying client.
func (a ServiceAccountWrap) SetDoAutomountServiceAccountToken(doAutomount bool) {
	var doAutomountPtrFromResource *bool = a.cache.AutomountServiceAccountToken
	if doAutomountPtrFromResource == nil || *doAutomountPtrFromResource != doAutomount {
		a.cache.AutomountServiceAccountToken = &doAutomount
	}
}

// Update performs an update of the service account resource object
// via the underlying client.
func (a *ServiceAccountWrap) Update(ctx context.Context) error {
	client := a.factory.CoreV1().ServiceAccounts(a.cache.GetNamespace())
	updatedObj, err := client.Update(ctx, a.cache, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	a.cache = updatedObj
	return nil
}

// AddRoleBinding creates a role binding in the targetNamespace connecting the service account with the specified cluster role
func (a *ServiceAccountWrap) AddRoleBinding(ctx context.Context, clusterRole RoleName, targetNamespace string) (*rbacv1.RoleBinding, error) {

	//Check if cluster role exists
	if checkRoleExistence {
		clusterRole, err := a.factory.RbacV1().ClusterRoles().Get(ctx, string(clusterRole), metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if clusterRole == nil {
			return nil, fmt.Errorf("Cluster Role '%v' does not exist", clusterRole)
		}
	}

	//Create role binding
	roleBindingClient := a.factory.RbacV1().RoleBindings(targetNamespace)
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(clusterRole),
			Namespace: targetNamespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      a.cache.GetName(),
				Namespace: a.cache.GetNamespace(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     string(clusterRole),
		},
	}
	return roleBindingClient.Create(ctx, roleBinding, metav1.CreateOptions{})
}

// GetServiceAccount returns *v1.ServiceAccount
func (a *ServiceAccountWrap) GetServiceAccount() *corev1.ServiceAccount {
	return a.cache
}

// ServiceAccountHelper implements functions to get service account secret
type ServiceAccountHelper interface {
	GetServiceAccountSecretNameRepeat(ctx context.Context) (string, error)
	GetServiceAccountSecretName(ctx context.Context) (string, error)
}
