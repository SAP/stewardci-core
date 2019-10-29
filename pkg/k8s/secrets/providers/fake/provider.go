package provider

import (
	"fmt"
	"github.com/SAP/stewardci-core/pkg/k8s/mocks"
	"github.com/SAP/stewardci-core/pkg/k8s/secrets"
	gomock "github.com/golang/mock/gomock"
	v1 "k8s.io/api/core/v1"
	"testing"
)

// NotExistingSecretName is a name of a not existing secret
const NotExistingSecretName = "notExistingSecret"

// ErrNotExisting is the error returned if not existing secret is requested
var ErrNotExisting = fmt.Errorf("secret not existing")

// NewProvider creates a fake secret provider for testing returning the secrets provided
func NewProvider(t *testing.T, namespace string, secrets ...*v1.Secret) secrets.SecretProvider {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockSecretProvider := mocks.NewMockSecretProvider(mockCtrl)
	for _, secret := range secrets {
		mockSecretProvider.EXPECT().GetSecret(secret.GetName()).Return(secret, nil).AnyTimes()
	}
	mockSecretProvider.EXPECT().GetSecret(NotExistingSecretName).Return(nil, ErrNotExisting).AnyTimes()
	return mockSecretProvider
}
