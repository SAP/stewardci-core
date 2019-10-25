package k8s

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
)

// AppendNameSuffix returns a mapping function from secret to secret
// in the result the secret has a new name with suffix 'suffix'
func AppendNameSuffix(suffix string) func(secret *v1.Secret) *v1.Secret {
	return func(secret *v1.Secret) *v1.Secret {
		secret.SetName(fmt.Sprintf("%s-%s", secret.GetName(), suffix))
		return secret
	}
}

// SetAnnotation returns a mapping function from secret to secret
// in the result secret the annotation with key 'key' is set to the value 'value'.
func SetAnnotation(key string, value string) func(secret *v1.Secret) *v1.Secret {
	return func(secret *v1.Secret) *v1.Secret {
		annotations := secret.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[key] = value
		secret.SetAnnotations(annotations)
		return secret
	}
}

// StripAnnotations returns a mapping function from secret to secret
// in the result secret all annotations with prefix 'keyPrefix' are removed.
func StripAnnotations(keyPrefix string) func(secret *v1.Secret) *v1.Secret {
	return func(secret *v1.Secret) *v1.Secret {
		annotations := secret.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		for key := range annotations {
			if strings.HasPrefix(key, keyPrefix) {
				delete(annotations, key)
			}
		}
		secret.SetAnnotations(annotations)
		return secret
	}
}

// SetLabel returns a mapping function from secret to secret
// in the result secret the label with key 'key' is set to the value 'value'.
func SetLabel(key string, value string) func(secret *v1.Secret) *v1.Secret {
	return func(secret *v1.Secret) *v1.Secret {
		labels := secret.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[key] = value
		secret.SetLabels(labels)
		return secret
	}
}

// StripLabels returns a mapping function from secret to secret
// in the result secret all labels with prefix 'keyPrefix' are removed.
func StripLabels(keyPrefix string) func(secret *v1.Secret) *v1.Secret {
	return func(secret *v1.Secret) *v1.Secret {
		labels := secret.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		for key := range labels {
			if strings.HasPrefix(key, keyPrefix) {
				delete(labels, key)
			}
		}
		secret.SetLabels(labels)
		return secret
	}
}
