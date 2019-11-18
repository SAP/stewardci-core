package secrets

import (
	"fmt"
	"strings"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	provider "github.com/SAP/stewardci-core/pkg/k8s/secrets/providers/fake"
	"github.com/pkg/errors"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	namespace       = "ns1"
	targetNamespace = "targetNs"
)

func initHelper(secrets ...*v1.Secret) (SecretHelper, corev1.SecretInterface) {
	provider := provider.NewProvider(namespace, secrets...)
	cf := fake.NewClientFactory()
	targetClient := cf.CoreV1().Secrets(targetNamespace)
	return NewSecretHelper(provider, targetNamespace, targetClient), targetClient
}

func Test_CopySecrets_NoFilter(t *testing.T) {
	t.Parallel()
	// SETUP
	helper, targetClient := initHelper(fake.SecretOpaque("foo", namespace))
	// EXERCISE
	list, err := helper.CopySecrets([]string{"foo"}, nil)
	// VERIFY
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"foo"}, list)
	storedSecret, _ := targetClient.Get("foo", metav1.GetOptions{})
	assert.Equal(t, "foo", storedSecret.GetName())
}

func nameStartsWithB(secret *v1.Secret) bool {
	return strings.HasPrefix(secret.GetName(), "b")
}

func Test_CopySecrets_WithFilter(t *testing.T) {
	t.Parallel()
	// SETUP
	helper, _ := initHelper(fake.SecretOpaque("foo", namespace),
		fake.SecretOpaque("bar", namespace),
		fake.SecretOpaque("baz", namespace))
	// EXERCISE
	list, err := helper.CopySecrets([]string{"foo", "bar", "baz"}, nameStartsWithB)
	// VERIFY
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"bar", "baz"}, list)
}

func Test_CopySecrets_NotExisting(t *testing.T) {
	t.Parallel()
	// SETUP
	helper, _ := initHelper(fake.SecretOpaque("foo", namespace),
		fake.SecretOpaque("bar", namespace))
	// EXERCISE
	list, err := helper.CopySecrets([]string{"foo", "notExistingSecret1", "bar"}, nil)
	// VERIFY
	assert.Assert(t, helper.IsNotFound(err))
	assert.DeepEqual(t, []string{"foo"}, list)
}

func Test_IsNotFound(t *testing.T) {
	t.Parallel()
	helper, _ := initHelper()
	err := NewNotFoundError("foo")
	assert.Assert(t, helper.IsNotFound(err))
}

func Test_IsNotFoundWrapped(t *testing.T) {
	t.Parallel()
	helper, _ := initHelper()
	err := NewNotFoundError("foo")
	err = errors.Wrap(err, "failed to copy secrets")
	assert.Assert(t, helper.IsNotFound(err))
}

func Test_IsNotFound_SameText(t *testing.T) {
	t.Parallel()
	helper, _ := initHelper()
	err := NewNotFoundError("foo")
	err = errors.WithMessage(nil, err.Error())
	assert.Assert(t, false == helper.IsNotFound(err))
}

func Test_IsNotFound_FmtError(t *testing.T) {
	t.Parallel()
	helper, _ := initHelper()
	err := fmt.Errorf("foo")
	assert.Assert(t, false == helper.IsNotFound(err))
}

func Test_IsNotFound_WithMessage(t *testing.T) {
	t.Parallel()
	helper, _ := initHelper()
	err := NewNotFoundError("foo")
	err = errors.WithMessage(err, "baz")
	assert.Assert(t, helper.IsNotFound(err))
}
