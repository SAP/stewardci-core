package secrets

import (
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"github.com/SAP/stewardci-core/pkg/k8s/secrets"
	provider "github.com/SAP/stewardci-core/pkg/k8s/secrets/providers/fake"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	namespace       = "ns1"
	targetNamespace = "targetNs"
)

func initHelper(t *testing.T) (SecretHelper, corev1.SecretInterface) {
	// SETUP
	provider := provider.NewProvider(t, namespace,
		fake.Secret("foo", namespace),
		fake.SecretWithType("docker1", namespace, v1.SecretTypeDockercfg),
		fake.SecretWithType("docker2", namespace, v1.SecretTypeDockerConfigJson))
	cf := fake.NewClientFactory()
	targetClient := cf.CoreV1().Secrets(targetNamespace)
	return NewSecretHelper(provider, targetNamespace, targetClient), targetClient
}
func Test_CopySecrets_NoFilter(t *testing.T) {
	helper, targetClient := initHelper(t)
	list, err := helper.CopySecrets([]string{"foo"}, nil)
	assert.NilError(t, err)
	assert.Equal(t, 1, len(list))
	assert.Equal(t, "foo", list[0])
	storedSecret, _ := targetClient.Get("foo", metav1.GetOptions{})
	assert.Equal(t, "foo", storedSecret.GetName(), "Name should be equal")
}

func Test_CopySecrets_MapName(t *testing.T) {
	helper, targetClient := initHelper(t)
	list, err := helper.CopySecrets([]string{"foo"}, nil, secrets.AppendNameSuffixFunc("suffix"))
	assert.NilError(t, err)
	assert.Equal(t, "foo-suffix", list[0])
	storedSecret, _ := targetClient.Get("foo-suffix", metav1.GetOptions{})
	assert.Equal(t, "foo-suffix", storedSecret.GetName(), "Name should be equal")
}

func Test_CopySecrets_DockerOnly(t *testing.T) {
	helper, _ := initHelper(t)
	list, err := helper.CopySecrets([]string{"foo", "docker1", "docker2"}, secrets.DockerOnly)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(list))
	assert.Equal(t, "docker1", list[0])
	assert.Equal(t, "docker2", list[1])
}

func Test_CopySecrets_NotExisting(t *testing.T) {
	helper, _ := initHelper(t)
	list, err := helper.CopySecrets([]string{"foo", provider.NotExistingSecretName, "docker1"}, nil)
	assert.Equal(t, provider.ErrNotExisting, err)
	assert.Equal(t, 1, len(list))
	assert.Equal(t, "foo", list[0])
}
