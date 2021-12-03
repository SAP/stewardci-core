package k8s

import (
	"context"
	"time"

	"github.com/SAP/stewardci-core/pkg/metrics"
	v1 "k8s.io/api/core/v1"
	errorsk8s "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klog "k8s.io/klog/v2"
)

type serviceAccountHelper struct {
	factory ClientFactory
	cache   *v1.ServiceAccount
}

// newServiceAccountHelper creates a new serviceAccountHelper
func newServiceAccountHelper(factory ClientFactory, cache *v1.ServiceAccount) *serviceAccountHelper {
	return &serviceAccountHelper{
		factory: factory,
		cache:   cache,
	}
}

// Reload performs an update of the cached service account resource object
// via the underlying client.
func (h *serviceAccountHelper) Reload(ctx context.Context) error {
	client := h.factory.CoreV1().ServiceAccounts(h.cache.GetNamespace())
	storedObj, err := client.Get(ctx, h.cache.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	h.cache = storedObj
	return nil
}

// GetServiceAccountSecretNameRepeat retrieves the name of the service account
// token secret.
// If no token is available, it retries until there is one.
func (h *serviceAccountHelper) GetServiceAccountSecretNameRepeat(ctx context.Context) (string, error) {
	retryInterval := 100 * time.Millisecond
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
				"namespace", h.cache.GetNamespace(),
				"serviceAccountName", h.cache.GetName(),
			)
		}
	}(time.Now())

	for {
		result, err := h.GetServiceAccountSecretName(ctx)
		if err != nil {
			return "", err
		}
		if result != "" {
			return result, nil
		}
		retryCount++
		time.Sleep(retryInterval)
		err = h.Reload(ctx)
		if err != nil {
			return "", err
		}
	}
}

// GetServiceAccountSecretName retrieves the name of the service account
// token secret.
func (h *serviceAccountHelper) GetServiceAccountSecretName(ctx context.Context) (string, error) {
	client := h.factory.CoreV1().Secrets(h.cache.GetNamespace())
	for _, secretRef := range h.cache.Secrets {
		secret, err := client.Get(ctx, secretRef.Name, metav1.GetOptions{})
		if err != nil {
			if errorsk8s.IsNotFound(err) {
				continue
			}
			return "", err
		}
		if secret.Type == v1.SecretTypeServiceAccountToken {
			return secret.Name, nil
		}
	}
	return "", nil
}
