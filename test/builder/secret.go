package builder

import (
	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretOp is an operation which modifies a Secret.
type SecretOp func(*v1.Secret)

// SecretBasicAuth creates a basic auth secret with jenkins.io credentilas-type annotation usernamePassword
func SecretBasicAuth(name, namespace, user, pwd string, ops ...SecretOp) *v1.Secret {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"jenkins.io/credentials-type": "usernamePassword"},
		},
		Type: v1.SecretTypeOpaque,
		StringData: map[string]string{"username": user,
			"password": pwd,
		},
	}
	for _, op := range ops {
		op(secret)
	}
	return secret
}

// SecretRename returns a SecretOp function which is adding a renaming annotation to a secret
func SecretRename(newName string) SecretOp {
	return func(secret *v1.Secret) {
		annotations := secret.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}
		annotations[api.AnnotationSecretRename] = newName
		secret.SetAnnotations(annotations)
	}
}
