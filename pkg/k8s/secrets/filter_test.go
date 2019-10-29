package secrets

import (
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func Test_DockerOnly_UntypedReturnsFalse(t *testing.T) {
	secret := fake.Secret("foo", "bar")
	assert.Assert(t, !DockerOnly(secret))
}

func Test_DockerOnly_TypeOpaqueReturnsFalse(t *testing.T) {
	secret := fake.SecretWithType("foo", "bar", v1.SecretTypeOpaque)
	assert.Assert(t, !DockerOnly(secret))
}

func Test_DockerOnly_DockerCfgReturnsTrue(t *testing.T) {
	secret := fake.SecretWithType("foo", "bar", v1.SecretTypeDockercfg)
	assert.Assert(t, DockerOnly(secret))
}

func Test_DockerOnly_DockerConfigJsonReturnsTrue(t *testing.T) {
	secret := fake.SecretWithType("foo", "bar", v1.SecretTypeDockerConfigJson)
	assert.Assert(t, DockerOnly(secret))
}
