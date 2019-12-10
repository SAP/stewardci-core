package builder

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretBasicAuth creates a basic auth secret with jenkins.io credentilas-type annotation usernamePassword
func SecretBasicAuth(name, namespace, user, pwd string) *v1.Secret {
	return &v1.Secret{
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
}
