package secrets

import (
	v1 "k8s.io/api/core/v1"
)

// SecretFilterType is a type for filter function
// true  -> keep item
// false -> skip item
// filter function nil keeps all items
type SecretFilterType = func(*v1.Secret) bool

// DockerOnly filter to filter only docker secrets
var DockerOnly SecretFilterType = func(secret *v1.Secret) bool {
	return secret.Type == v1.SecretTypeDockerConfigJson || secret.Type == v1.SecretTypeDockercfg
}
