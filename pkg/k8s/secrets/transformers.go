package secrets

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"strings"
)

// SecretTransformerType is a type for secret transformers
type SecretTransformerType = func(*v1.Secret) *v1.Secret

// AppendNameSuffixFunc returns a mapping function from secret to secret
// in the result the secret has a new name with suffix 'suffix'
func AppendNameSuffixFunc(suffix string) SecretTransformerType {
	return func(secret *v1.Secret) *v1.Secret {
		secret.SetName(fmt.Sprintf("%s-%s", secret.GetName(), suffix))
		return secret
	}
}

// SetAnnotationFunc returns a mapping function from secret to secret
// in the result secret the annotation with key 'key' is set to the value 'value'.
func SetAnnotationFunc(key string, value string) SecretTransformerType {
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

// StripAnnotationsFunc returns a mapping function from secret to secret
// in the result secret all annotations with prefix 'keyPrefix' are removed.
func StripAnnotationsFunc(keyPrefix string) SecretTransformerType {
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

// SetLabelFunc returns a mapping function from secret to secret
// in the result secret the label with key 'key' is set to the value 'value'.
func SetLabelFunc(key string, value string) SecretTransformerType {
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

// StripLabelsFunc returns a mapping function from secret to secret
// in the result secret all labels with prefix 'keyPrefix' are removed.
func StripLabelsFunc(keyPrefix string) SecretTransformerType {
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
