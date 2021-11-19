package k8s

import (
	"time"

	"github.com/SAP/stewardci-core/pkg/metrics"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klog "k8s.io/klog/v2"
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
	retryCount := uint64(0)
	defer func(start time.Time) {
		if retryCount > 0 {
			codeLocationSkipFrames := uint16(1)
			codeLocation := metrics.CodeLocation(codeLocationSkipFrames)
			latency := time.Since(start)
			metrics.Retries.Observe(codeLocation, retryCount, latency)
			klog.V(5).InfoS("retry was required",
				"location", codeLocation,
				"count", retryCount,
				"latency", latency,
				"namespace", a.cache.GetNamespace(),
				"serviceAccountName", a.cache.GetName(),
			)
		}
	}(time.Now())
	duration, _ := time.ParseDuration("100ms")
	for {
		result := a.GetServiceAccountSecretName()
		if result != "" {
			return result
		}
		retryCount++
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
