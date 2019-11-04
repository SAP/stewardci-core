package secrets

import (
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	provider "github.com/SAP/stewardci-core/pkg/k8s/secrets/providers/fake"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"testing"
)

const (
	namespace       = "ns1"
	targetNamespace = "targetNs"
)

func initHelper(secrets ...*v1.Secret) (SecretHelper, corev1.SecretInterface) {
	// SETUP
	provider := provider.NewProvider(namespace, secrets...)
	cf := fake.NewClientFactory()
	targetClient := cf.CoreV1().Secrets(targetNamespace)
	return NewSecretHelper(provider, targetNamespace, targetClient), targetClient
}

func Test_CopySecrets_NoFilter(t *testing.T) {
	helper, targetClient := initHelper(fake.Secret("foo", namespace))
	list, err := helper.CopySecrets([]string{"foo"}, nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"foo"}, list)
	storedSecret, _ := targetClient.Get("foo", metav1.GetOptions{})
	assert.Equal(t, "foo", storedSecret.GetName())
}

func Test_CopySecrets_DockerOnly(t *testing.T) {
	helper, _ := initHelper(fake.Secret("foo", namespace),
		fake.SecretWithType("docker1", namespace, v1.SecretTypeDockercfg),
		fake.SecretWithType("docker2", namespace, v1.SecretTypeDockerConfigJson))
	list, err := helper.CopySecrets([]string{"foo", "docker1", "docker2"}, DockerOnly)
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"docker1", "docker2"}, list)
}

func Test_CopySecrets_NotExisting(t *testing.T) {
	helper, _ := initHelper(fake.Secret("foo", namespace),
		fake.SecretWithType("docker1", namespace, v1.SecretTypeDockercfg))
	list, err := helper.CopySecrets([]string{"foo", "notExistingSecret1", "docker1"}, nil)
	assert.Assert(t, err != nil)
	assert.DeepEqual(t, []string{"foo"}, list)
}
