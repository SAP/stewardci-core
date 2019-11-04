package secrets

import (
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func Test_DockerOnly_UntypedReturnsFalse(t *testing.T) {
	secret := fake.Secret("foo", "bar")
	result := DockerOnly(secret)
	assert.Assert(t, result == false)
}

func Test_DockerOnly_TypeOpaqueReturnsFalse(t *testing.T) {
	secret := fake.SecretWithType("foo", "bar", v1.SecretTypeOpaque)
	result := DockerOnly(secret)
	assert.Assert(t, result == false)
}

func Test_DockerOnly_DockerCfgReturnsTrue(t *testing.T) {
	secret := fake.SecretWithType("foo", "bar", v1.SecretTypeDockercfg)
	result := DockerOnly(secret)
	assert.Assert(t, result == true)
}

func Test_DockerOnly_DockerConfigJsonReturnsTrue(t *testing.T) {
	secret := fake.SecretWithType("foo", "bar", v1.SecretTypeDockerConfigJson)
	result := DockerOnly(secret)
	assert.Assert(t, result == true)
}
