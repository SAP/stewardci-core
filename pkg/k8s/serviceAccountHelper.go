package k8s

import (
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klog "k8s.io/klog/v2"
)

type serviceAccountHelper struct {
	factory                ClientFactory
	cache                  *v1.ServiceAccount
	durationByTypeObserver DurationByTypeObserver
}

//newServiceAccountHelper creates ServiceAccountManager
func newServiceAccountHelper(factory ClientFactory, cache *v1.ServiceAccount, durationByTypeObserver DurationByTypeObserver) *serviceAccountHelper {
	return &serviceAccountHelper{
		factory:                factory,
		cache:                  cache,
		durationByTypeObserver: durationByTypeObserver,
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
	var isRetry bool
	defer func(start time.Time) {
		if isRetry && a.durationByTypeObserver != nil {
			elapsed := time.Since(start)
			klog.V(5).Infof("retry of service acount secret retrieving took %v for %s/%s", elapsed, a.cache.GetNamespace(), a.cache.GetName())
			a.durationByTypeObserver.ObserveRetryDurationByType("RunNamespaceServiceAccountSecretCreation", elapsed)
		}
	}(time.Now())
	duration, _ := time.ParseDuration("100ms")
	for {
		result := a.GetServiceAccountSecretName()
		if result != "" {
			return result
		}
		isRetry = true
		time.Sleep(duration)
		a.Reload()
	}
}

// GetServiceAccountSecretName returns the name of the default-token of the service account
func (a *serviceAccountHelper) GetServiceAccountSecretName() string {
	for _, secretRef := range a.cache.Secrets {
		client := a.factory.CoreV1().Secrets(a.cache.GetNamespace())
		secret, err := client.Get(secretRef.Name, metav1.GetOptions{})
		if err == nil &&
			secret != nil &&
			secret.Type == v1.SecretTypeServiceAccountToken {
			return secret.Name
		}
	}
	return ""
}
