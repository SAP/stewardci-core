package secrets

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	secretMocks "github.com/SAP/stewardci-core/pkg/k8s/secrets/mocks"
	fakesecretprovider "github.com/SAP/stewardci-core/pkg/k8s/secrets/providers/fake"
	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubernetes "k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	namespace       = "ns1"
	targetNamespace = "targetNs"
)

func initSecretHelperWithMock(
	t *testing.T, mockCtrl *gomock.Controller, secrets ...*v1.Secret,
) (
	SecretHelper, *secretMocks.MockSecretHelper,
) {
	t.Helper()
	fakeSecretProvider := fakesecretprovider.NewProvider(namespace, secrets...)

	cf := fake.NewClientFactory()
	targetClient := cf.CoreV1().Secrets(targetNamespace)

	mockSecretHelper := secretMocks.NewMockSecretHelper(mockCtrl)
	helper := NewSecretHelper(fakeSecretProvider, targetNamespace, targetClient).(*secretHelper)
	helper.testing = &secretHelperTesting{
		createSecretStub: mockSecretHelper.CreateSecret,
	}
	return helper, mockSecretHelper
}

func Test_CopySecrets_NoFilter(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	secret := fake.SecretOpaque("foo", namespace)
	examinee, mockSecretHelper := initSecretHelperWithMock(t, mockCtrl, secret)

	// EXPECT
	expectedSecret := fake.SecretOpaque("foo", "")
	mockSecretHelper.EXPECT().CreateSecret(ctx, expectedSecret).Return(expectedSecret, nil)

	// EXERCISE
	resultList, resultErr := examinee.CopySecrets(ctx, []string{"foo"}, nil)

	// VERIFY
	assert.NilError(t, resultErr)
	assert.DeepEqual(t, []string{"foo"}, resultList)
}

func Test_CopySecrets_WithFilter(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	examinee, mockSecretHelper := initSecretHelperWithMock(t, mockCtrl,
		fake.SecretOpaque("foo", namespace),
		fake.SecretOpaque("bar", namespace),
		fake.SecretOpaque("foo2", namespace),
		fake.SecretOpaque("baz", namespace),
		fake.SecretOpaque("foo3", namespace),
	)

	filter := func(secret *v1.Secret) bool {
		return strings.HasPrefix(secret.GetName(), "b")
	}

	// EXPECT
	expectedSecret2 := fake.SecretOpaque("bar", "")
	mockSecretHelper.EXPECT().CreateSecret(ctx, expectedSecret2).Return(expectedSecret2, nil)
	expectedSecret3 := fake.SecretOpaque("baz", "")
	mockSecretHelper.EXPECT().CreateSecret(ctx, expectedSecret3).Return(expectedSecret3, nil)

	// EXERCISE
	resultList, resultErr := examinee.CopySecrets(
		ctx,
		[]string{"foo", "bar", "foo2", "baz", "foo3"},
		filter,
	)

	// VERIFY
	assert.NilError(t, resultErr)
	assert.DeepEqual(t, []string{"bar", "baz"}, resultList)
}

func Test_CopySecrets_NotExisting(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	examinee, mockSecretHelper := initSecretHelperWithMock(t, mockCtrl,
		fake.SecretOpaque("foo", namespace),
		fake.SecretOpaque("bar", namespace),
	)

	// EXPECT
	expectedSecret := fake.SecretOpaque("foo", "")
	mockSecretHelper.EXPECT().CreateSecret(ctx, expectedSecret).Return(expectedSecret, nil)

	// EXERCISE
	resultList, resultErr := examinee.CopySecrets(
		ctx,
		[]string{"foo", "notExistingSecret1", "bar"},
		nil,
	)

	// VERIFY
	assert.Assert(t, examinee.IsNotFound(resultErr))
	assert.DeepEqual(t, []string{"foo"}, resultList)
}

func initSecretHelperWithClient(secrets ...*v1.Secret) (SecretHelper, corev1.SecretInterface) {
	provider := fakesecretprovider.NewProvider(namespace, secrets...)
	cf := fake.NewClientFactory()
	targetClient := cf.CoreV1().Secrets(targetNamespace)
	return NewSecretHelper(provider, targetNamespace, targetClient), targetClient
}

func Test_CreateSecret_GoodCase(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	examinee, targetClient := initSecretHelperWithClient()
	origSecret := fake.SecretOpaque("foo", namespace)

	// EXERCISE
	resultSecret, resultErr := examinee.CreateSecret(ctx, origSecret.DeepCopy())

	// VERIFY
	expectedSecret := origSecret.DeepCopy()
	expectedSecret.SetNamespace(targetNamespace)

	assert.NilError(t, resultErr)
	assert.DeepEqual(t, expectedSecret, resultSecret)

	storedSecret, err := targetClient.Get(ctx, "foo", metav1.GetOptions{})
	assert.NilError(t, err)
	assert.DeepEqual(t, expectedSecret, storedSecret)
}

func Test_CreateSecret_StripsMetadata(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	examinee, targetClient := initSecretHelperWithClient()

	now := metav1.Now()
	var grace int64 = 1
	origSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "foo",
			GenerateName:               "dummy",
			Namespace:                  "ns1",
			SelfLink:                   "dummy",
			UID:                        types.UID("dummy"),
			ResourceVersion:            "dummy",
			Generation:                 1,
			CreationTimestamp:          now,
			DeletionGracePeriodSeconds: &grace,
			OwnerReferences:            []metav1.OwnerReference{{}},
			Finalizers:                 []string{"dummy"},
			Labels: map[string]string{
				"lbar": "lbaz",
			},
			Annotations: map[string]string{
				"abar": "abaz",
			},
		},
		Type: v1.SecretTypeOpaque,
	}

	// EXERCISE
	resultSecret, resultErr := examinee.CreateSecret(ctx, origSecret.DeepCopy())

	// VERIFY
	expectedSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:         origSecret.GetName(),
			GenerateName: origSecret.GetGenerateName(),
			Namespace:    targetNamespace,
			Labels:       origSecret.GetLabels(),
			Annotations:  origSecret.GetAnnotations(),
		},
		Type: v1.SecretTypeOpaque,
	}

	assert.NilError(t, resultErr)
	assert.DeepEqual(t, expectedSecret, resultSecret)

	storedSecret, err := targetClient.Get(ctx, "foo", metav1.GetOptions{})
	assert.NilError(t, err)
	assert.DeepEqual(t, expectedSecret, storedSecret)
}

func Test_CreateSecret_Error(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	fakeSecretProvider := fakesecretprovider.NewProvider(namespace)
	cs := kubernetes.NewSimpleClientset()
	expectedError := fmt.Errorf("expected")
	cs.PrependReactor("create", "*", fake.NewErrorReactor(expectedError))

	origSecret := fake.SecretOpaque("foo", namespace)
	examinee := NewSecretHelper(fakeSecretProvider, targetNamespace, cs.CoreV1().Secrets(targetNamespace))

	// EXERCISE
	resultSecret, resultErr := examinee.CreateSecret(ctx, origSecret.DeepCopy())

	// VERIFY
	assert.Assert(t, expectedError == resultErr)
	assert.Assert(t, resultSecret == nil)
}

func Test_IsNotFound_True(t *testing.T) {
	t.Parallel()

	for ti, notFoundError := range []error{
		// direct
		NewNotFoundError("foo"),
		// cause
		errors.Wrap(NewNotFoundError("foo"), "bar"),
		// cause of cause
		errors.Wrap(errors.Wrap(NewNotFoundError("foo"), "bar"), "baz"),
		// cause
		errors.WithMessage(NewNotFoundError("foo"), "bar"),
	} {
		t.Run(strconv.Itoa(ti), func(t *testing.T) {
			// SETUP
			examinee, _ := initSecretHelperWithClient()

			// EXERCISE
			result := examinee.IsNotFound(notFoundError)

			// VERIFY
			assert.Assert(t, result == true)
		})
	}
}

func Test_IsNotFound_False(t *testing.T) {
	t.Parallel()

	for ti, otherError := range []error{
		// same message as NFE:
		errors.New(NewNotFoundError("foo").Error()),
		fmt.Errorf("foo"),
	} {
		t.Run(strconv.Itoa(ti), func(t *testing.T) {
			// SETUP
			examinee, _ := initSecretHelperWithClient()

			// EXERCISE
			result := examinee.IsNotFound(otherError)

			// VERIFY
			assert.Assert(t, result == false)
		})
	}
}
