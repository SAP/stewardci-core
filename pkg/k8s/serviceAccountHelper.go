package k8s

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceAccountTokenSecretRetriever retrieves the service account
// token secret for a service account.
type ServiceAccountTokenSecretRetriever interface {
	// ForServiceAccount retrieves ... for the given service account object
	// using the K8s client in `ctx`.
	ForObj(ctx context.Context, serviceAccount *v1.ServiceAccount) (v1.Secret, error)

	// ForName retries ... for the named service account
	// using the K8s client in `ctx`.
	ForName(ctx context.Context, serviceAccountName, namespace string) (v1.Secret, error)
}

// EnsureServiceAccountTokenSecretRetriever sets the
// `ServiceAccountTokenSecretRetriever` in the given context,
// if not present.
// The returned context either is or extends the given context and
// has a `ServiceAccountTokenSecretRetriever` instance stored.
func EnsureServiceAccountTokenSecretRetrieverFromContext(ctx context.Context) context.Context {
	instance := GetServiceAccountTokenSecretRetrieverFromContext(ctx)
	if instance == nil {
		instance = &serviceAccountTokenSecretRetrieverImpl{}
		return WithServiceAccountTokenSecretRetriever(ctx, instance)
	}
	return ctx
}

// --- default impl ---

type serviceAccountTokenSecretRetrieverImpl struct{}

func (r *serviceAccountTokenSecretRetrieverImpl) ForObj(ctx context.Context, serviceAccount *v1.ServiceAccount) (v1.Secret, error) {
	return r.ForName(ctx, serviceAccount.GetName(), serviceAccount.GetNamespace())
}

func (r *serviceAccountTokenSecretRetrieverImpl) ForName(ctx context.Context, serviceAccountName, namespace string) (v1.Secret, error) {
	factory := GetClientFactory(ctx)
	client := factory.CoreV1().ServiceAccounts(namespace)
	serviceAccount, err := client.Get(serviceAccountName, metav1.GetOptions{})
	if err != nil {
	return	r.forNameRetry(ctx, serviceAccountName, namespace)
	}

	result := r.GetServiceAccountSecret(ctx, serviceAccount)
	if result != nil {
		return *result, nil
	}
	return r.forNameRetry(ctx, serviceAccountName, namespace)
}

func (r *serviceAccountTokenSecretRetrieverImpl) forNameRetry(ctx context.Context, serviceAccountName, namespace string) (v1.Secret, error) {
	duration, _ := time.ParseDuration("100ms")
	time.Sleep(duration)
	return r.ForName(ctx, serviceAccountName, namespace)
}

// GetServiceAccountSecret returns the default-token of the service account
func (r *serviceAccountTokenSecretRetrieverImpl) GetServiceAccountSecret(ctx context.Context, serviceAccount *v1.ServiceAccount) *v1.Secret {
	factory := GetClientFactory(ctx)
	for _, secretRef := range serviceAccount.Secrets {
		client := factory.CoreV1().Secrets(secretRef.Namespace)
		secret, err := client.Get(secretRef.Name, metav1.GetOptions{})
		if err == nil &&
			secret != nil &&
			secret.Type == v1.SecretTypeServiceAccountToken {
			return secret
		}
	}
	return nil
}
