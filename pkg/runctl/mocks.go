package runctl

import (
	"context"

	stewardfake "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/fake"
	"github.com/SAP/stewardci-core/pkg/k8s"
	fake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	"github.com/SAP/stewardci-core/pkg/k8s/secrets"
	secretMocks "github.com/SAP/stewardci-core/pkg/k8s/secrets/mocks"
	tektonclientfake "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned/fake"
	gomock "github.com/golang/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func mockFactories(ctx context.Context, ctrl *gomock.Controller) context.Context {
	mockFactory := mocks.NewMockClientFactory(ctrl)

	kubeClientSet := kubefake.NewSimpleClientset()
	kubeClientSet.PrependReactor("create", "*", fake.GenerateNameReactor(0))

	mockFactory.EXPECT().CoreV1().Return(kubeClientSet.CoreV1()).AnyTimes()
	mockFactory.EXPECT().RbacV1beta1().Return(kubeClientSet.RbacV1beta1()).AnyTimes()
	mockFactory.EXPECT().NetworkingV1().Return(kubeClientSet.NetworkingV1()).AnyTimes()

	dynamicClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	mockFactory.EXPECT().Dynamic().Return(dynamicClient).AnyTimes()

	stewardClientset := stewardfake.NewSimpleClientset()
	mockFactory.EXPECT().StewardV1alpha1().Return(stewardClientset.StewardV1alpha1()).AnyTimes()

	tektonClientset := tektonclientfake.NewSimpleClientset()
	mockFactory.EXPECT().TektonV1alpha1().Return(tektonClientset.TektonV1alpha1()).AnyTimes()

	ctx = k8s.WithClientFactory(ctx, mockFactory)

	namespaceManager := k8s.NewNamespaceManager(mockFactory, runNamespacePrefix, runNamespaceRandomLength)
	ctx = k8s.WithNamespaceManager(ctx, namespaceManager)

	mockSecretProvider := secretMocks.NewMockSecretProvider(ctrl)
	ctx = secrets.WithSecretProvider(ctx, mockSecretProvider)

	//ctx = WithRunInstanceTesting(ctx, newRunManagerTestingWithAllNoopStubs())

	return ctx
}
