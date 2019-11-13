package secrets

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"strings"
)

// SecretTransformerType is a type for secret transformers
// the function MUST NOT modify the original secret but
// return a copy of the given one even if no modification took place
type SecretTransformerType = func(*v1.Secret) *v1.Secret

// UniqueNameTransformer returns a transforming function from secret to secret
// the resulting secret has generateName set to the original name plus '-' as separator
// and name is unset
func UniqueNameTransformer() SecretTransformerType {
	return func(secret *v1.Secret) *v1.Secret {
		secret = secret.DeepCopy()
		secret.SetGenerateName(fmt.Sprintf("%s-", secret.GetName()))
		secret.SetName("")
		return secret
	}
}

// SetAnnotationTransformer returns a transforming function from secret to secret
// in the result secret the annotation with key 'key' is set to the value 'value'.
func SetAnnotationTransformer(key string, value string) SecretTransformerType {
	return func(secret *v1.Secret) *v1.Secret {
		secret = secret.DeepCopy()
		annotations := secret.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[key] = value
		secret.SetAnnotations(annotations)
		return secret
	}
}

// StripAnnotationsTransformer returns a transforming function from secret to secret
// in the result secret all annotations with prefix 'keyPrefix' are removed.
func StripAnnotationsTransformer(keyPrefix string) SecretTransformerType {
	return func(secret *v1.Secret) *v1.Secret {
		secret = secret.DeepCopy()
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

// SetLabelTransformer returns a transforming function from secret to secret
// in the result secret the label with key 'key' is set to the value 'value'.
func SetLabelTransformer(key string, value string) SecretTransformerType {
	return func(secret *v1.Secret) *v1.Secret {
		secret = secret.DeepCopy()
		labels := secret.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[key] = value
		secret.SetLabels(labels)
		return secret
	}
}

// StripLabelsTransformer returns a transforming function from secret to secret
// in the result secret all labels with prefix 'keyPrefix' are removed.
func StripLabelsTransformer(keyPrefix string) SecretTransformerType {
	return func(secret *v1.Secret) *v1.Secret {
		secret = secret.DeepCopy()
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
