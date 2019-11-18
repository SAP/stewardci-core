package secrets

import (
	"fmt"
	"strings"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	secretMocks "github.com/SAP/stewardci-core/pkg/k8s/secrets/mocks"
	provider "github.com/SAP/stewardci-core/pkg/k8s/secrets/providers/fake"
	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	namespace       = "ns1"
	targetNamespace = "targetNs"
)

func initHelperWithMock(t *testing.T, secrets ...*v1.Secret) (SecretHelper, *secretMocks.MockSecretHelper) {
	t.Helper()
	provider := provider.NewProvider(namespace, secrets...)

	cf := fake.NewClientFactory()
	targetClient := cf.CoreV1().Secrets(targetNamespace)

	mockCtrl := gomock.NewController(t)
	mockSecretHelper := secretMocks.NewMockSecretHelper(mockCtrl)
	helper := NewSecretHelper(provider, targetNamespace, targetClient)
	x := helper.(*secretHelper)
	x.testing = &secretHelperTesting{createSecretStub: mockSecretHelper.CreateSecret}
	return helper, mockSecretHelper
}

func Test_CopySecrets_NoFilter(t *testing.T) {
	t.Parallel()
	// SETUP
	secret := fake.SecretOpaque("foo", namespace)
	helper, mockSecretHelper := initHelperWithMock(t, secret)
	expectedSecret := fake.SecretOpaque("foo", "")
	// VERIFY
	mockSecretHelper.EXPECT().CreateSecret(expectedSecret).Return(expectedSecret, nil)
	// EXERCISE
	list, err := helper.CopySecrets([]string{"foo"}, nil)
	// VERIFY
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"foo"}, list)

}

func nameStartsWithB(secret *v1.Secret) bool {
	return strings.HasPrefix(secret.GetName(), "b")
}

func Test_CopySecrets_WithFilter(t *testing.T) {
	t.Parallel()
	// SETUP
	helper, mockSecretHelper := initHelperWithMock(t,
		fake.SecretOpaque("foo", namespace),
		fake.SecretOpaque("bar", namespace),
		fake.SecretOpaque("baz", namespace),
	)
	expectedSecret1 := fake.SecretOpaque("foo", "")
	expectedSecret2 := fake.SecretOpaque("bar", "")
	expectedSecret3 := fake.SecretOpaque("baz", "")
	// VERIFY
	mockSecretHelper.EXPECT().CreateSecret(expectedSecret1).Return(expectedSecret1, nil)
	mockSecretHelper.EXPECT().CreateSecret(expectedSecret2).Return(expectedSecret2, nil)
	mockSecretHelper.EXPECT().CreateSecret(expectedSecret3).Return(expectedSecret3, nil)
	// EXERCISE
	list, err := helper.CopySecrets([]string{"foo", "bar", "baz"}, nameStartsWithB)
	// VERIFY
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"bar", "baz"}, list)
}

func Test_CopySecrets_NotExisting(t *testing.T) {
	t.Parallel()
	// SETUP
	helper, mockSecretHelper := initHelperWithMock(t,
		fake.SecretOpaque("foo", namespace),
		fake.SecretOpaque("bar", namespace),
	)
	expectedSecret := fake.SecretOpaque("foo", "")
	// VERIFY
	mockSecretHelper.EXPECT().CreateSecret(expectedSecret).Return(expectedSecret, nil)
	// EXERCISE
	list, err := helper.CopySecrets([]string{"foo", "notExistingSecret1", "bar"}, nil)
	// VERIFY
	assert.Assert(t, helper.IsNotFound(err))
	assert.DeepEqual(t, []string{"foo"}, list)
}

func initHelperWithClient(secrets ...*v1.Secret) (SecretHelper, corev1.SecretInterface) {
	provider := provider.NewProvider(namespace, secrets...)
	cf := fake.NewClientFactory()
	targetClient := cf.CoreV1().Secrets(targetNamespace)
	return NewSecretHelper(provider, targetNamespace, targetClient), targetClient
}

func Test_CreateSecret(t *testing.T) {
	t.Parallel()
	// SETUP
	helper, targetClient := initHelperWithClient()
	secret := fake.SecretOpaque("foo", namespace)
	// EXERCISE
	_, err := helper.CreateSecret(secret)
	// VERIFY
	assert.NilError(t, err)
	storedSecret, err := targetClient.Get("foo", metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Equal(t, "foo", storedSecret.GetName())
}

func Test_CreateSecret_Error(t *testing.T) {
	t.Parallel()
	// SETUP
	provider := provider.NewProvider(namespace)
	cs := kubernetes.NewSimpleClientset()
	expectedError := fmt.Errorf("expected")
	cs.PrependReactor("create", "*", fake.NewErrorReactor(expectedError))
	helper := NewSecretHelper(provider, targetNamespace, cs.CoreV1().Secrets(targetNamespace))
	secret := fake.SecretOpaque("foo", namespace)

	// EXERCISE
	_, err := helper.CreateSecret(secret)
	// VERIFY
	assert.Assert(t, expectedError == err)
}

func Test_IsNotFound(t *testing.T) {
	t.Parallel()
	helper, _ := initHelperWithClient()
	err := NewNotFoundError("foo")
	assert.Assert(t, helper.IsNotFound(err))
}

func Test_IsNotFoundWrapped(t *testing.T) {
	t.Parallel()
	helper, _ := initHelperWithClient()
	err := NewNotFoundError("foo")
	err = errors.Wrap(err, "failed to copy secrets")
	assert.Assert(t, helper.IsNotFound(err))
}

func Test_IsNotFound_SameText(t *testing.T) {
	t.Parallel()
	helper, _ := initHelperWithClient()
	err := NewNotFoundError("foo")
	err = errors.WithMessage(nil, err.Error())
	assert.Assert(t, false == helper.IsNotFound(err))
}

func Test_IsNotFound_FmtError(t *testing.T) {
	t.Parallel()
	helper, _ := initHelperWithClient()
	err := fmt.Errorf("foo")
	assert.Assert(t, false == helper.IsNotFound(err))
}

func Test_IsNotFound_WithMessage(t *testing.T) {
	t.Parallel()
	helper, _ := initHelperWithClient()
	err := NewNotFoundError("foo")
	err = errors.WithMessage(err, "baz")
	assert.Assert(t, helper.IsNotFound(err))
}
