package secrets

import (
	"fmt"

	"strings"

	v1 "k8s.io/api/core/v1"
)

// SecretTransformer is a function that modifies the given secret.
type SecretTransformer = func(*v1.Secret)

// UniqueNameTransformer returns a secret transformer function that sets
// `metadata.generateName` to the original `metadata.name` with '-' appended as
// separator and removes `metadata.name`.
func UniqueNameTransformer() SecretTransformer {
	return func(secret *v1.Secret) {
		secret.SetGenerateName(fmt.Sprintf("%s-", secret.GetName()))
		secret.SetName("")
	}
}

// RenameByAttributeTransformer returns a secret transformer function that sets
// `metadata.name` to the name defined in `metadata.annotations` with the defined key
// no rename is done if the annotation with the key does not exist
// no rename is done if an empty key is provided
// no rename is done if the value of the annotation with the defined key is the empty string
func RenameByAttributeTransformer(key string) SecretTransformer {
	return func(secret *v1.Secret) {
		annotations := secret.GetAnnotations()
		if annotations != nil && annotations[key] != "" {
			secret.SetName(annotations[key])
		}
	}
}

// SetAnnotationTransformer returns a secret transformer function that sets the
// annotation with the given key to the given value.
func SetAnnotationTransformer(key string, value string) SecretTransformer {
	return func(secret *v1.Secret) {
		annotations := secret.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[key] = value
		secret.SetAnnotations(annotations)
	}
}

// StripAnnotationsTransformer returns a secret transformer function that
// removes all annotations where the key starts with the given 'keyPrefix'.
func StripAnnotationsTransformer(keyPrefix string) SecretTransformer {
	return func(secret *v1.Secret) {
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
	}
}

// SetLabelTransformer returns a secret transformer function that sets the
// label with the given key to the given value.
func SetLabelTransformer(key string, value string) SecretTransformer {
	return func(secret *v1.Secret) {
		labels := secret.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[key] = value
		secret.SetLabels(labels)
	}
}

// StripLabelsTransformer returns a secret transformer function that
// removes all labels where the key starts with the given 'keyPrefix'.
func StripLabelsTransformer(keyPrefix string) SecretTransformer {
	return func(secret *v1.Secret) {
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
	}
}
