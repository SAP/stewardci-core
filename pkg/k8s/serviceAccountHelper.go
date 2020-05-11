package k8s

import (
	"log"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type serviceAccountHelper struct {
	factory ClientFactory
	cache   *v1.ServiceAccount
}

//newServiceAccountHelper creates ServiceAccountManager
func newServiceAccountHelper(factory ClientFactory, cache *v1.ServiceAccount) *serviceAccountHelper {
	return &serviceAccountHelper{
		factory: factory,
		cache:   cache,
	}
}

// Reload performs an update of the cached service account resource object
// via the underlying client.
func (a *serviceAccountHelper) Reload() error {
	client := a.factory.CoreV1().ServiceAccounts(a.cache.GetNamespace())
	storedObj, err := client.Get(a.cache.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	a.cache = storedObj
	return nil
}

// GetServiceAccountSecretNameRepeat returns the name of the default-token of the service account
func (a *serviceAccountHelper) GetServiceAccountSecretNameRepeat() string {
	duration, _ := time.ParseDuration("100ms")
	for {
		result := a.GetServiceAccountSecretName()
		if result != "" {
			return result
		}
		time.Sleep(duration)
		a.Reload()
	}
}

// GetServiceAccountSecretName returns the name of the default-token of the service account
func (a *serviceAccountHelper) GetServiceAccountSecretName() string {
	log.Printf("ServiceAccount: %+v", a.cache)
	for _, secretRef := range a.cache.Secrets {
		client := a.factory.CoreV1().Secrets(secretRef.Namespace)
		secret, err := client.Get(secretRef.Name, metav1.GetOptions{})
		log.Printf("Secret: %+v %+v", err, secret)
		if err == nil &&
			secret != nil &&
			secret.Type == v1.SecretTypeServiceAccountToken {
			return secret.Name
		}
	}
	return ""
}
