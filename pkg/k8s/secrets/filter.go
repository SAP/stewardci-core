package secrets

import (
	v1 "k8s.io/api/core/v1"
)

// SecretFilter is a type for filter function
// true  -> keep item
// false -> skip item
// filter function nil keeps all items
type SecretFilter = func(*v1.Secret) bool

// check that signature conforms to type
var _ SecretFilter = DockerOnly

// DockerOnly selects only secrets of type `kubernetes.io/dockerconfigjson` and `kubernetes.io/dockercfg`.
func DockerOnly(secret *v1.Secret) bool {
	return secret.Type == v1.SecretTypeDockerConfigJson || secret.Type == v1.SecretTypeDockercfg
}
