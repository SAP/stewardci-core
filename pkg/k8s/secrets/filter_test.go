package secrets

import (
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/v3/assert"
	v1 "k8s.io/api/core/v1"
)

func Test_DockerOnly(t *testing.T) {
	t.Parallel()
	type tests struct {
		secretType     v1.SecretType
		expectedResult bool
	}
	testSet := []tests{
		{secretType: v1.SecretTypeOpaque, expectedResult: false},
		{secretType: v1.SecretTypeServiceAccountToken, expectedResult: false},
		{secretType: v1.SecretTypeBasicAuth, expectedResult: false},
		{secretType: v1.SecretTypeSSHAuth, expectedResult: false},
		{secretType: v1.SecretTypeTLS, expectedResult: false},
		{secretType: v1.SecretTypeDockercfg, expectedResult: true},
		{secretType: v1.SecretTypeDockerConfigJson, expectedResult: true},
	}
	for _, test := range testSet {
		secret := fake.SecretWithType("foo", "bar", test.secretType)
		result := DockerOnly(secret)
		assert.Assert(t, result == test.expectedResult)
	}
}
