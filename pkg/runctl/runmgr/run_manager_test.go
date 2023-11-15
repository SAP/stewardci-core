package runmgr

import (
	"context"
	"fmt"
	"testing"

	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	serrors "github.com/SAP/stewardci-core/pkg/errors"
	featureflag "github.com/SAP/stewardci-core/pkg/featureflag"
	featureflagtesting "github.com/SAP/stewardci-core/pkg/featureflag/testing"
	k8s "github.com/SAP/stewardci-core/pkg/k8s"
	fake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	k8sfake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	k8smocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	"github.com/SAP/stewardci-core/pkg/k8s/secrets"
	secretproviderfakes "github.com/SAP/stewardci-core/pkg/k8s/secrets/providers/fake"
	k8ssecretprovider "github.com/SAP/stewardci-core/pkg/k8s/secrets/providers/k8s"
	cfg "github.com/SAP/stewardci-core/pkg/runctl/cfg"
	"github.com/SAP/stewardci-core/pkg/runctl/constants"
	runifc "github.com/SAP/stewardci-core/pkg/runctl/run"
	runmocks "github.com/SAP/stewardci-core/pkg/runctl/run/mocks"
	runctltesting "github.com/SAP/stewardci-core/pkg/runctl/testing"
	"github.com/SAP/stewardci-core/pkg/utils"
	spew "github.com/davecgh/go-spew/spew"
	gomock "github.com/golang/mock/gomock"
	errors "github.com/pkg/errors"
	tektonPod "github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	assert "gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	equality "k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func newRunManagerTestingWithAllNoopStubs() *runManagerTesting {
	return &runManagerTesting{
		cleanupStub:                               func(context.Context, *runContext) error { return nil },
		copySecretsToRunNamespaceStub:             func(context.Context, *runContext) (string, []string, error) { return "", []string{}, nil },
		setupLimitRangeFromConfigStub:             func(context.Context, *runContext) error { return nil },
		setupNetworkPolicyFromConfigStub:          func(context.Context, *runContext) error { return nil },
		setupNetworkPolicyThatIsolatesAllPodsStub: func(context.Context, *runContext) error { return nil },
		setupResourceQuotaFromConfigStub:          func(context.Context, *runContext) error { return nil },
		setupServiceAccountStub:                   func(context.Context, *runContext, string, []string) error { return nil },
		setupStaticLimitRangeStub:                 func(context.Context, *runContext) error { return nil },
		setupStaticNetworkPoliciesStub:            func(context.Context, *runContext) error { return nil },
		setupStaticResourceQuotaStub:              func(context.Context, *runContext) error { return nil },
	}
}

func newRunManagerTestingWithRequiredStubs() *runManagerTesting {
	return &runManagerTesting{}
}

func contextWithSpec(t *testing.T, runNamespaceName string, spec stewardv1alpha1.PipelineSpec) *runContext {
	ctx := context.Background()
	pipelineRun := k8sfake.PipelineRun("run1", "ns1", spec)
	k8sPipelineRun, err := k8s.NewPipelineRun(ctx, pipelineRun, nil)
	assert.NilError(t, err)
	return &runContext{runNamespace: runNamespaceName,
		pipelineRun: k8sPipelineRun,
	}
}

func Test__runManager_prepareRunNamespace__CreatesNamespaces(t *testing.T) {
	for _, ffEnabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("featureflag_CreateAuxNamespaceIfUnused_%t", ffEnabled), func(t *testing.T) {
			defer featureflagtesting.WithFeatureFlag(featureflag.CreateAuxNamespaceIfUnused, ffEnabled)()

			// SETUP
			h := newTestHelper1(t)

			cf := newFakeClientFactory(
				k8sfake.Namespace(h.namespace1),
				k8sfake.PipelineRun(h.pipelineRun1, h.namespace1, stewardv1alpha1.PipelineSpec{}),
			)
			cf.KubernetesClientset().PrependReactor("create", "namespaces", k8sfake.GenerateNameReactor(7))

			config := &cfg.PipelineRunsConfigStruct{}
			secretProvider := secretproviderfakes.NewProvider(h.namespace1)

			examinee := NewRunManager(cf, secretProvider).(*runManager)
			examinee.testing = newRunManagerTestingWithAllNoopStubs()

			pipelineRunHelper, err := k8s.NewPipelineRun(h.ctx, h.getPipelineRunFromStorage(cf, h.namespace1, h.pipelineRun1), cf)
			assert.NilError(t, err)
			runCtx := &runContext{
				pipelineRun:        pipelineRunHelper,
				pipelineRunsConfig: config,
			}

			// EXERCISE
			resultErr := examinee.prepareRunNamespace(h.ctx, runCtx)

			// VERIFY
			assert.NilError(t, resultErr)

			// namespaces
			{
				pipelineRun1 := h.getPipelineRunFromStorage(cf, h.namespace1, h.pipelineRun1)
				expectedNamespaces := []string{h.namespace1}

				h.VerifyNamespace(cf, runCtx.runNamespace, "main", runNamespaceRandomLength)
				expectedNamespaces = append(expectedNamespaces, runCtx.runNamespace)

				if ffEnabled {
					h.VerifyNamespace(cf, runCtx.auxNamespace, "aux", runNamespaceRandomLength)
					expectedNamespaces = append(expectedNamespaces, runCtx.auxNamespace)
				} else {
					assert.Equal(t, pipelineRun1.Status.AuxiliaryNamespace, "")
				}

				h.assertThatExactlyTheseNamespacesExist(cf, expectedNamespaces...)
			}
		})
	}
}

func Test__runManager_prepareRunNamespace__Calls__copySecretsToRunNamespace__AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)

	cf := newFakeClientFactory(
		k8sfake.Namespace(h.namespace1),
		k8sfake.PipelineRun(h.pipelineRun1, h.namespace1, stewardv1alpha1.PipelineSpec{}),
	)

	config := &cfg.PipelineRunsConfigStruct{}
	secretProvider := secretproviderfakes.NewProvider(h.namespace1)
	pipelineRunHelper, err := k8s.NewPipelineRun(h.ctx, h.getPipelineRunFromStorage(cf, h.namespace1, h.pipelineRun1), cf)
	assert.NilError(t, err)

	examinee := NewRunManager(cf, secretProvider).(*runManager)
	examinee.testing = newRunManagerTestingWithAllNoopStubs()

	expectedError := errors.New("some error")
	var methodCalled bool
	examinee.testing.copySecretsToRunNamespaceStub = func(_ context.Context, runCtx *runContext) (string, []string, error) {
		methodCalled = true
		assert.Assert(t, runCtx.pipelineRun == pipelineRunHelper)
		assert.Assert(t, runCtx.runNamespace != "")
		return "", nil, expectedError
	}

	runCtx := &runContext{
		pipelineRun:        pipelineRunHelper,
		pipelineRunsConfig: config,
	}

	// EXERCISE
	resultErr := examinee.prepareRunNamespace(h.ctx, runCtx)

	// VERIFY
	assert.Equal(t, expectedError, resultErr)
	assert.Assert(t, methodCalled == true)
}

func Test__runManager_prepareRunNamespace__Calls_setupServiceAccount_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)

	cf := newFakeClientFactory(
		k8sfake.Namespace(h.namespace1),
		k8sfake.PipelineRun(h.pipelineRun1, h.namespace1, stewardv1alpha1.PipelineSpec{}),
	)

	config := &cfg.PipelineRunsConfigStruct{}
	secretProvider := secretproviderfakes.NewProvider(h.namespace1)
	pipelineRunHelper, err := k8s.NewPipelineRun(h.ctx, h.getPipelineRunFromStorage(cf, h.namespace1, h.pipelineRun1), cf)
	assert.NilError(t, err)

	examinee := NewRunManager(cf, secretProvider).(*runManager)
	examinee.testing = newRunManagerTestingWithAllNoopStubs()

	expectedPipelineCloneSecretName := "pipelineCloneSecret1"
	expectedImagePullSecretNames := []string{"imagePullSecret1"}
	expectedError := errors.New("some error")
	var methodCalled bool
	examinee.testing.setupServiceAccountStub = func(_ context.Context, runCtx *runContext, pipelineCloneSecretName string, imagePullSecretNames []string) error {
		methodCalled = true
		assert.Assert(t, runCtx.runNamespace != "")
		assert.Equal(t, expectedPipelineCloneSecretName, pipelineCloneSecretName)
		assert.DeepEqual(t, expectedImagePullSecretNames, imagePullSecretNames)
		return expectedError
	}
	examinee.testing.copySecretsToRunNamespaceStub = func(_ context.Context, runCtx *runContext) (string, []string, error) {
		return expectedPipelineCloneSecretName, expectedImagePullSecretNames, nil
	}

	runCtx := &runContext{
		pipelineRun:        pipelineRunHelper,
		pipelineRunsConfig: config,
	}

	// EXERCISE
	resultErr := examinee.prepareRunNamespace(h.ctx, runCtx)

	// VERIFY
	assert.Equal(t, expectedError, resultErr)
	assert.Assert(t, methodCalled == true)
}

func Test__runManager_prepareRunNamespace__Calls_setupStaticNetworkPolicies_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)

	cf := newFakeClientFactory(
		k8sfake.Namespace(h.namespace1),
		k8sfake.PipelineRun(h.pipelineRun1, h.namespace1, stewardv1alpha1.PipelineSpec{}),
	)

	config := &cfg.PipelineRunsConfigStruct{}
	secretProvider := secretproviderfakes.NewProvider(h.namespace1)
	pipelineRunHelper, err := k8s.NewPipelineRun(h.ctx, h.getPipelineRunFromStorage(cf, h.namespace1, h.pipelineRun1), cf)
	assert.NilError(t, err)

	examinee := NewRunManager(cf, secretProvider).(*runManager)
	examinee.testing = newRunManagerTestingWithAllNoopStubs()

	expectedError := errors.New("some error")
	var methodCalled bool
	examinee.testing.setupStaticNetworkPoliciesStub = func(_ context.Context, runCtx *runContext) error {
		methodCalled = true
		assert.Assert(t, runCtx.runNamespace != "")
		return expectedError
	}

	runCtx := &runContext{
		pipelineRun:        pipelineRunHelper,
		pipelineRunsConfig: config,
	}

	// EXERCISE
	resultErr := examinee.prepareRunNamespace(h.ctx, runCtx)

	// VERIFY
	assert.Equal(t, expectedError, resultErr)
	assert.Assert(t, methodCalled == true)
}

func Test__runManager_setupStaticNetworkPolicies__Succeeds(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	runCtx := &runContext{}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticNetworkPoliciesStub = nil

	// EXERCISE
	resultError := examinee.setupStaticNetworkPolicies(ctx, runCtx)

	// VERIFY
	assert.NilError(t, resultError)
}

func Test__runManager_setupStaticNetworkPolicies__Calls_setupNetworkPolicyThatIsolatesAllPods_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := &runContext{runNamespace: h.namespace1}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticNetworkPoliciesStub = nil

	var methodCalled bool
	expectedError := errors.New("some error")
	examinee.testing.setupNetworkPolicyThatIsolatesAllPodsStub = func(_ context.Context, ctx *runContext) error {
		methodCalled = true
		assert.Equal(t, h.namespace1, ctx.runNamespace)
		return expectedError
	}

	// EXERCISE
	resultError := examinee.setupStaticNetworkPolicies(h.ctx, runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to set up the network policy isolating all pods in namespace \""+h.namespace1+"\": ")
	assert.Assert(t, errors.Cause(resultError) == expectedError)
	assert.Assert(t, methodCalled == true)
}

func Test__runManager_setupStaticNetworkPolicies__Calls_setupNetworkPolicyFromConfig_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := &runContext{runNamespace: h.namespace1}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticNetworkPoliciesStub = nil

	var methodCalled bool
	expectedError := errors.New("some error")
	examinee.testing.setupNetworkPolicyFromConfigStub = func(_ context.Context, ctx *runContext) error {
		methodCalled = true
		assert.Equal(t, h.namespace1, ctx.runNamespace)
		return expectedError
	}

	// EXERCISE
	resultError := examinee.setupStaticNetworkPolicies(h.ctx, runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to set up the configured network policy in namespace \""+h.namespace1+"\": ")
	assert.Assert(t, errors.Cause(resultError) == expectedError)
	assert.Assert(t, methodCalled == true)
}

func Test__runManager_setupNetworkPolicyThatIsolatesAllPods(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		expectedNamePrefix = "steward.sap.com--isolate-all-"
	)
	h := newTestHelper1(t)
	runCtx := &runContext{runNamespace: h.namespace1}
	cf := k8sfake.NewClientFactory()
	cf.KubernetesClientset().PrependReactor("create", "*", k8sfake.GenerateNameReactor(0))

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyThatIsolatesAllPodsStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyThatIsolatesAllPods(h.ctx, runCtx)
	assert.NilError(t, resultError)

	// VERIFY
	actualPolicies, err := cf.NetworkingV1().NetworkPolicies(h.namespace1).List(h.ctx, metav1.ListOptions{})
	assert.NilError(t, err)
	assert.Assert(t, len(actualPolicies.Items) == 1)
	{
		policy := actualPolicies.Items[0]
		assert.Equal(t, expectedNamePrefix, policy.GetName())
		assert.DeepEqual(t, policy.GetLabels(),
			map[string]string{
				stewardv1alpha1.LabelSystemManaged: "",
			},
		)
	}
}

func Test__runManager_setupNetworkPolicyFromConfig__NoPolicyConfigured(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := contextWithSpec(t, h.namespace1, stewardv1alpha1.PipelineSpec{})
	runCtx.pipelineRunsConfig = &cfg.PipelineRunsConfigStruct{
		// no network policy
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if no policy is configured.
	cf := k8smocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(h.ctx, runCtx)

	// VERIFY
	assert.NilError(t, resultError)
}

func Test__runManager_setupNetworkPolicyFromConfig__SetsMetadataAndLeavesOtherThingsUntouched(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		expectedNamePrefix = "steward.sap.com--configured-"
	)
	h := newTestHelper1(t)

	gvr := schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v123",
		Resource: "networkpolicies",
	}

	runCtx := contextWithSpec(t, h.namespace1, stewardv1alpha1.PipelineSpec{})
	runCtx.pipelineRunsConfig = &cfg.PipelineRunsConfigStruct{
		DefaultNetworkProfile: "key1",
		NetworkPolicies: map[string]string{
			"key1": fixIndent(`
				apiVersion: networking.k8s.io/v123
				kind: NetworkPolicy
				# no metadata here
				customStuffNotTouchedByController:
					a: 1
					b: true
				spec:
					# bogus spec to check if SUT modifies something
					undefinedField: [42, 17]
					podSelector: true
					namespaceSelector: false
					policyTypes:
					-	undefinedKey
					egress:
						undefinedField: string1
					ingress:
						undefinedField: string1
				`),
		},
	}

	cf := k8sfake.NewClientFactory()
	cf.DynamicClient = dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			gvr: "NetworkPolicyList",
		},
	)
	cf.DynamicClient.PrependReactor("create", "*", k8sfake.GenerateNameReactor(0))

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(h.ctx, runCtx)
	assert.NilError(t, resultError)

	// VERIFY
	actualPolicies, err := cf.Dynamic().Resource(gvr).List(h.ctx, metav1.ListOptions{})
	assert.NilError(t, err)
	assert.Equal(t, 1, len(actualPolicies.Items))
	{
		policy := actualPolicies.Items[0]

		expectedMetadata := map[string]interface{}{
			"name":         expectedNamePrefix,
			"generateName": expectedNamePrefix,
			"namespace":    h.namespace1,
			"labels": map[string]interface{}{
				stewardv1alpha1.LabelSystemManaged: "",
			},
		}
		assert.DeepEqual(t, expectedMetadata, policy.Object["metadata"])

		expectedCustomStuff := map[string]interface{}{
			"a": int64(1),
			"b": true,
		}
		assert.DeepEqual(t, expectedCustomStuff, policy.Object["customStuffNotTouchedByController"])

		expectedSpec := map[string]interface{}{
			"undefinedField":    []interface{}{int64(42), int64(17)},
			"podSelector":       true,
			"namespaceSelector": false,
			"policyTypes":       []interface{}{"undefinedKey"},
			"egress":            map[string]interface{}{"undefinedField": "string1"},
			"ingress":           map[string]interface{}{"undefinedField": "string1"},
		}
		assert.DeepEqual(t, expectedSpec, policy.Object["spec"])
	}
}

func Test__runManager_setupNetworkPolicyFromConfig__ReplacesAllMetadata(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		expectedNamePrefix = "steward.sap.com--configured-"
	)
	h := newTestHelper1(t)
	runCtx := contextWithSpec(t, h.namespace1, stewardv1alpha1.PipelineSpec{})

	gvr := schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v123",
		Resource: "networkpolicies",
	}

	runCtx.pipelineRunsConfig = &cfg.PipelineRunsConfigStruct{
		DefaultNetworkProfile: "key1",
		NetworkPolicies: map[string]string{
			"key1": fixIndent(`
				apiVersion: networking.k8s.io/v123
				kind: NetworkPolicy
				metadata:
					name: name1
					generateName: generateName1
					labels:
						label1: labelVal1
						label2: labelVal2
					annotations:
						annotation1: annotationVal1
						annotation2: annotationVal2
					creationTimestamp: "2000-01-01T00:00:00Z"
					generation: 99999
					resourceVersion: "12345678"
					selfLink: /foo/bar
					uid: 00000000-0000-0000-0000-000000000000
					finalizers:
						- finalizer1
					undefinedField: "abc"
				`),
		},
	}

	cf := k8sfake.NewClientFactory()
	cf.DynamicClient = dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			gvr: "NetworkPolicyList",
		},
	)
	cf.DynamicClient.PrependReactor("create", "*", k8sfake.GenerateNameReactor(0))

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(h.ctx, runCtx)
	assert.NilError(t, resultError)

	// VERIFY
	actualPolicies, err := cf.Dynamic().Resource(gvr).List(h.ctx, metav1.ListOptions{})
	assert.NilError(t, err)
	assert.Equal(t, 1, len(actualPolicies.Items))
	{
		policy := actualPolicies.Items[0]
		expectedMetadata := map[string]interface{}{
			"name":         expectedNamePrefix,
			"generateName": expectedNamePrefix,
			"namespace":    h.namespace1,
			"labels": map[string]interface{}{
				stewardv1alpha1.LabelSystemManaged: "",
			},
		}
		assert.DeepEqual(t, expectedMetadata, policy.Object["metadata"])
	}
}

func Test_RunManager_setupNetworkPolicyFromConfig_ChooseCorrectPolicy(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name           string
		profilesSpec   *stewardv1alpha1.Profiles
		expectedPolicy string
		expectError    bool
		result         stewardv1alpha1.Result
	}{
		{
			name:           "no_profile_spec",
			profilesSpec:   nil,
			expectedPolicy: "networkPolicySpecDefault1",
			expectError:    false,
			result:         stewardv1alpha1.ResultUndefined,
		},
		{
			name:           "no_network_profile",
			profilesSpec:   &stewardv1alpha1.Profiles{},
			expectedPolicy: "networkPolicySpecDefault1",
			expectError:    false,
			result:         stewardv1alpha1.ResultUndefined,
		},
		{
			name: "undefined_network_profile",
			profilesSpec: &stewardv1alpha1.Profiles{
				Network: "undefined1",
			},
			expectError: true,
			result:      stewardv1alpha1.ResultErrorConfig,
		},
		{
			name: "network_profile_1",
			profilesSpec: &stewardv1alpha1.Profiles{
				Network: "networkPolicyKey1",
			},
			expectedPolicy: "networkPolicySpec1",
			expectError:    false,
			result:         stewardv1alpha1.ResultUndefined,
		},
		{
			name: "network_profile_2",
			profilesSpec: &stewardv1alpha1.Profiles{
				Network: "networkPolicyKey2",
			},
			expectedPolicy: "networkPolicySpec2",
			expectError:    false,
			result:         stewardv1alpha1.ResultUndefined,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			t.Parallel()

			// SETUP
			ctx := context.Background()
			gvr := schema.GroupVersionResource{
				Group:    "networking.k8s.io",
				Version:  "v123",
				Resource: "networkpolicies",
			}
			cf := k8sfake.NewClientFactory()
			cf.DynamicClient = dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
				runtime.NewScheme(),
				map[schema.GroupVersionResource]string{
					gvr: "NetworkPolicyList",
				},
			)
			cf.DynamicClient.PrependReactor("create", "*", k8sfake.GenerateNameReactor(0))

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockPipelineRun := k8smocks.NewMockPipelineRun(mockCtrl)
			mockPipelineRun.EXPECT().
				GetSpec().
				Return(&stewardv1alpha1.PipelineSpec{Profiles: tc.profilesSpec}).
				AnyTimes()

			runCtx := &runContext{pipelineRun: mockPipelineRun}

			runCtx.pipelineRunsConfig = &cfg.PipelineRunsConfigStruct{
				DefaultNetworkProfile: "networkPolicyKey0",
				NetworkPolicies: map[string]string{
					"networkPolicyKey0": fixIndent(`
						apiVersion: networking.k8s.io/v123
						kind: NetworkPolicy
						spec: networkPolicySpecDefault1`),
					"networkPolicyKey1": fixIndent(`
						apiVersion: networking.k8s.io/v123
						kind: NetworkPolicy
						spec: networkPolicySpec1`),
					"networkPolicyKey2": fixIndent(`
						apiVersion: networking.k8s.io/v123
						kind: NetworkPolicy
						spec: networkPolicySpec2`),
				},
			}

			examinee := runManager{
				factory: cf,
				testing: newRunManagerTestingWithAllNoopStubs(),
			}
			examinee.testing.setupNetworkPolicyFromConfigStub = nil

			// EXERCISE
			resultError := examinee.setupNetworkPolicyFromConfig(ctx, runCtx)

			// VERIFY
			if tc.expectError {
				assert.Assert(t, resultError != nil)
				if tc.result != stewardv1alpha1.ResultUndefined {
					assert.Equal(t, tc.result, serrors.GetClass(resultError))
				}
			} else {
				assert.NilError(t, resultError)
				actualPolicies, err := cf.Dynamic().Resource(gvr).List(ctx, metav1.ListOptions{})

				assert.NilError(t, err)
				assert.Equal(t, 1, len(actualPolicies.Items))
				{
					policy := actualPolicies.Items[0]
					assert.DeepEqual(t, tc.expectedPolicy, policy.Object["spec"])
				}
			}
		})
	}
}

func Test_RunManager_setupNetworkPolicyFromConfig_MalformedPolicy(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := contextWithSpec(t, h.namespace1, stewardv1alpha1.PipelineSpec{})
	runCtx.pipelineRunsConfig = &cfg.PipelineRunsConfigStruct{
		DefaultNetworkProfile: "key1",
		NetworkPolicies: map[string]string{
			"key1": ":", // malformed YAML
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := k8smocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(h.ctx, runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to decode configured network policy: ")
}

func Test__runManager_setupNetworkPolicyFromConfig__UnexpectedGroup(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := contextWithSpec(t, h.namespace1, stewardv1alpha1.PipelineSpec{})
	runCtx.pipelineRunsConfig = &cfg.PipelineRunsConfigStruct{
		DefaultNetworkProfile: "key1",
		NetworkPolicies: map[string]string{
			"key1": fixIndent(`
				apiVersion: unexpected.group/v1
				kind: NetworkPolicy
				`),
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := k8smocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(h.ctx, runCtx)

	// VERIFY
	assert.Error(t, resultError,
		"configured network policy does not denote a"+
			" \"NetworkPolicy.networking.k8s.io\" but a"+
			" \"NetworkPolicy.unexpected.group\"")
}

func Test__runManager_setupNetworkPolicyFromConfig__UnexpectedKind(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := contextWithSpec(t, h.namespace1, stewardv1alpha1.PipelineSpec{})
	runCtx.pipelineRunsConfig = &cfg.PipelineRunsConfigStruct{
		DefaultNetworkProfile: "key1",
		NetworkPolicies: map[string]string{
			"key1": fixIndent(`
				apiVersion: networking.k8s.io/v1
				kind: UnexpectedKind
				`),
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := k8smocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(h.ctx, runCtx)

	// VERIFY
	assert.Error(t, resultError,
		"configured network policy does not denote a"+
			" \"NetworkPolicy.networking.k8s.io\" but a"+
			" \"UnexpectedKind.networking.k8s.io\"")
}

func Test__runManager_setupStaticLimitRange__Calls__setupLimitRangeFromConfig__AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := &runContext{runNamespace: h.namespace1}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticLimitRangeStub = nil

	var methodCalled bool
	expectedError := errors.New("some error")
	examinee.testing.setupLimitRangeFromConfigStub = func(_ context.Context, ctx *runContext) error {
		methodCalled = true
		assert.Equal(t, h.namespace1, ctx.runNamespace)
		return expectedError
	}

	// EXERCISE
	resultError := examinee.setupStaticLimitRange(h.ctx, runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to set up the configured limit range in namespace \""+h.namespace1+"\": ")
	assert.Assert(t, errors.Cause(resultError) == expectedError)
	assert.Assert(t, methodCalled == true)
}

func Test__runManager_setupStaticLimitRange__Succeeds(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	runCtx := &runContext{}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticLimitRangeStub = nil

	// EXERCISE
	resultError := examinee.setupStaticLimitRange(ctx, runCtx)

	// VERIFY
	assert.NilError(t, resultError)
}

func Test__runManager_setupLimitRangeFromConfig__NoLimitRangeConfigured(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := &runContext{
		runNamespace: h.namespace1,
		pipelineRunsConfig: &cfg.PipelineRunsConfigStruct{
			LimitRange: "", // no policy
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if no policy is configured.
	cf := k8smocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupLimitRangeFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupLimitRangeFromConfig(h.ctx, runCtx)
	assert.NilError(t, resultError)
}

func Test__runManager_setupLimitRangeFromConfig__MalformedLimitRange(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := &runContext{
		runNamespace: h.namespace1,
		pipelineRunsConfig: &cfg.PipelineRunsConfigStruct{
			LimitRange: ":", // malformed YAML
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := k8smocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupLimitRangeFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupLimitRangeFromConfig(h.ctx, runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to decode configured limit range: ")
}

func Test__runManager_setupLimitRangeFromConfig__UnexpectedGroup(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := &runContext{
		runNamespace: h.namespace1,
		pipelineRunsConfig: &cfg.PipelineRunsConfigStruct{
			LimitRange: fixIndent(`
                                apiVersion: unexpected.group/v1
                                kind: LimitRange
                                `),
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := k8smocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupLimitRangeFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupLimitRangeFromConfig(h.ctx, runCtx)

	// VERIFY
	assert.Error(t, resultError,
		"configured limit range does not denote a"+
			" \"LimitRange\" but a"+
			" \"LimitRange.unexpected.group\"")
}

func Test__runManager_setupLimitRangeFromConfig__UnexpectedKind(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := &runContext{
		runNamespace: h.namespace1,
		pipelineRunsConfig: &cfg.PipelineRunsConfigStruct{
			LimitRange: fixIndent(`
                                apiVersion: v1
                                kind: UnexpectedKind
                                `),
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := k8smocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupLimitRangeFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupLimitRangeFromConfig(h.ctx, runCtx)

	// VERIFY
	assert.Error(t, resultError,
		"configured limit range does not denote a"+
			" \"LimitRange\" but a"+
			" \"UnexpectedKind\"")
}

func Test__runManager_setupStaticResourceQuota__Calls__setupResourceQuotaFromConfig__AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := &runContext{runNamespace: h.namespace1}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticResourceQuotaStub = nil

	var methodCalled bool
	expectedError := errors.New("some error")
	examinee.testing.setupResourceQuotaFromConfigStub = func(_ context.Context, ctx *runContext) error {
		methodCalled = true
		assert.Equal(t, h.namespace1, ctx.runNamespace)
		return expectedError
	}

	// EXERCISE
	resultError := examinee.setupStaticResourceQuota(h.ctx, runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to set up the configured resource quota in namespace \""+h.namespace1+"\": ")
	assert.Assert(t, errors.Cause(resultError) == expectedError)
	assert.Assert(t, methodCalled == true)
}

func Test__runManager_setupStaticResourceQuota__Succeeds(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	runCtx := &runContext{}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticResourceQuotaStub = nil

	// EXERCISE
	resultError := examinee.setupStaticResourceQuota(ctx, runCtx)

	// VERIFY
	assert.NilError(t, resultError)
}

func Test__runManager_setupResourceQuotaFromConfig__NoQuotaConfigured(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := &runContext{
		runNamespace: h.namespace1,
		pipelineRunsConfig: &cfg.PipelineRunsConfigStruct{
			LimitRange: "", // no policy
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if no policy is configured.
	cf := k8smocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupResourceQuotaFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupResourceQuotaFromConfig(h.ctx, runCtx)
	assert.NilError(t, resultError)
}

func Test__runManager_setupResourceQuotaFromConfig__MalformedResourceQuota(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := &runContext{
		runNamespace: h.namespace1,
		pipelineRunsConfig: &cfg.PipelineRunsConfigStruct{
			ResourceQuota: ":", // malformed YAML
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := k8smocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupResourceQuotaFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupResourceQuotaFromConfig(h.ctx, runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to decode configured resource quota: ")
}

func Test__runManager_setupResourceQuotaFromConfig__UnexpectedGroup(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := &runContext{
		runNamespace: h.namespace1,
		pipelineRunsConfig: &cfg.PipelineRunsConfigStruct{
			ResourceQuota: fixIndent(`
                                apiVersion: unexpected.group/v1
                                kind: ResourceQuota
                                `),
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := k8smocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupResourceQuotaFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupResourceQuotaFromConfig(h.ctx, runCtx)

	// VERIFY
	assert.Error(t, resultError,
		"configured resource quota does not denote a"+
			" \"ResourceQuota\" but a"+
			" \"ResourceQuota.unexpected.group\"")
}

func Test__runManager_setupResourceQuotaFromConfig__UnexpectedKind(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	runCtx := &runContext{
		runNamespace: h.namespace1,
		pipelineRunsConfig: &cfg.PipelineRunsConfigStruct{
			ResourceQuota: fixIndent(`
                                apiVersion: v1
                                kind: UnexpectedKind
                                `),
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := k8smocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupResourceQuotaFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupResourceQuotaFromConfig(h.ctx, runCtx)

	// VERIFY
	assert.Error(t, resultError,
		"configured resource quota does not denote a"+
			" \"ResourceQuota\" but a"+
			" \"UnexpectedKind\"")
}

func Test__runManager_createTektonTaskRun__PodTemplate_IsNotEmptyIfNoValuesToSet(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	_, mockPipelineRun, _ := h.prepareMocks(mockCtrl)
	runConfig := h.runsConfigWithTaskData()
	runCtx := &runContext{
		pipelineRun:        mockPipelineRun,
		pipelineRunsConfig: runConfig,
		runNamespace:       h.namespace1,
	}
	mockPipelineRun.UpdateRunNamespace(h.namespace1)
	cf := k8sfake.NewClientFactory()
	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}

	// EXERCISE
	resultError := examinee.createTektonTaskRun(h.ctx, runCtx)

	// VERIFY
	assert.NilError(t, resultError)

	taskRun, err := cf.TektonV1beta1().TaskRuns(h.namespace1).Get(h.ctx, constants.TektonTaskRunName, metav1.GetOptions{})
	assert.NilError(t, err)
	if equality.Semantic.DeepEqual(taskRun.Spec.PodTemplate, tektonPod.PodTemplate{}) {
		t.Fatal("podTemplate of TaskRun is empty")
	}
}

func Test__runManager_createTektonTaskRun__PodTemplate_AllValuesSet(t *testing.T) {
	t.Parallel()

	int64Ptr := func(val int64) *int64 { return &val }

	// SETUP
	h := newTestHelper1(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	_, mockPipelineRun, _ := h.prepareMocks(mockCtrl)
	mockPipelineRun.UpdateRunNamespace(h.namespace1)
	runConfig := h.runsConfigWithTaskData()
	runConfig.Timeout = utils.Metav1Duration(4444)
	runConfig.JenkinsfileRunnerPodSecurityContextFSGroup = int64Ptr(1111)
	runConfig.JenkinsfileRunnerPodSecurityContextRunAsGroup = int64Ptr(2222)
	runConfig.JenkinsfileRunnerPodSecurityContextRunAsUser = int64Ptr(3333)
	runCtx := &runContext{
		pipelineRun:        mockPipelineRun,
		runNamespace:       h.namespace1,
		pipelineRunsConfig: runConfig,
	}
	cf := k8sfake.NewClientFactory()

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}

	// EXERCISE
	resultError := examinee.createTektonTaskRun(h.ctx, runCtx)

	// VERIFY
	assert.NilError(t, resultError)

	taskRun, err := cf.TektonV1beta1().TaskRuns(h.namespace1).Get(h.ctx, constants.TektonTaskRunName, metav1.GetOptions{})
	assert.NilError(t, err)
	automount := true

	expectedPodTemplate := &tektonPod.PodTemplate{
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup:    int64Ptr(1111),
			RunAsGroup: int64Ptr(2222),
			RunAsUser:  int64Ptr(3333),
		},
		AutomountServiceAccountToken: &automount,
	}
	podTemplate := taskRun.Spec.PodTemplate
	assert.DeepEqual(t, expectedPodTemplate, podTemplate)
	assert.Assert(t, podTemplate.SecurityContext.FSGroup != runCtx.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextFSGroup)
	assert.Assert(t, podTemplate.SecurityContext.RunAsGroup != runCtx.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextRunAsGroup)
	assert.Assert(t, podTemplate.SecurityContext.RunAsUser != runCtx.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextRunAsUser)
	assert.DeepEqual(t, utils.Metav1Duration(4444), taskRun.Spec.Timeout)
}

func Test__runManager_addTektonTaskRunParamsForLoggingElasticsearch(t *testing.T) {
	t.Parallel()
	const (
		TaskRunParamNameIndexURL  = "PIPELINE_LOG_ELASTICSEARCH_INDEX_URL"
		TaskRunParamNameRunIDJSON = "PIPELINE_LOG_ELASTICSEARCH_RUN_ID_JSON"
		DummyRunID                = "runID1"
		SampleURL                 = "http://foo.bar/baz"
	)
	idParam := tektonStringParam(TaskRunParamNameRunIDJSON, fmt.Sprintf("%q", DummyRunID))

	for _, test := range []struct {
		name           string
		spec           *stewardv1alpha1.PipelineSpec
		expectedParams tektonv1beta1.Params
		expectedError  error
	}{
		{
			name: "url in pipeline",
			spec: &stewardv1alpha1.PipelineSpec{
				Logging: &stewardv1alpha1.Logging{
					Elasticsearch: &stewardv1alpha1.Elasticsearch{
						RunID:    &stewardv1alpha1.CustomJSON{Value: DummyRunID},
						IndexURL: SampleURL,
					},
				},
			},
			expectedParams: tektonv1beta1.Params{
				idParam,
				tektonStringParam(TaskRunParamNameIndexURL, SampleURL),
			},
		},
		{
			name: "no url in pipeline",
			spec: &stewardv1alpha1.PipelineSpec{
				Logging: &stewardv1alpha1.Logging{
					Elasticsearch: &stewardv1alpha1.Elasticsearch{
						RunID: &stewardv1alpha1.CustomJSON{Value: DummyRunID},
					},
				},
			},
			expectedParams: tektonv1beta1.Params{
				idParam,
			},
		},
		{
			name: "no logging configured will set url to empty string",
			spec: &stewardv1alpha1.PipelineSpec{},
			expectedParams: tektonv1beta1.Params{
				tektonStringParam(TaskRunParamNameIndexURL, ""),
			},
		},
		{
			name: "empty logging configured will set url to empty string",
			spec: &stewardv1alpha1.PipelineSpec{
				Logging: &stewardv1alpha1.Logging{},
			},
			expectedParams: tektonv1beta1.Params{
				tektonStringParam(TaskRunParamNameIndexURL, ""),
			},
		},
		{
			name: "wrong url returns error",
			spec: &stewardv1alpha1.PipelineSpec{
				Logging: &stewardv1alpha1.Logging{
					Elasticsearch: &stewardv1alpha1.Elasticsearch{
						RunID:    &stewardv1alpha1.CustomJSON{Value: DummyRunID},
						IndexURL: "wrongscheme://foo.bar",
					},
				},
			},
			expectedError: fmt.Errorf(`field "spec.logging.elasticsearch.indexURL" has invalid value "wrongscheme://foo.bar": scheme not supported: "wrongscheme"`),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// SETUP
			h := newTestHelper1(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockFactory, mockPipelineRun, mockSecretProvider := h.prepareMocksWithSpec(mockCtrl, test.spec)

			examinee := NewRunManager(mockFactory, mockSecretProvider).(*runManager)
			examinee.testing = newRunManagerTestingWithRequiredStubs()

			runCtx := &runContext{
				pipelineRun: mockPipelineRun,
			}

			tektonTaskRun := &tektonv1beta1.TaskRun{}

			// EXERCISE
			resultError := examinee.addTektonTaskRunParamsForLoggingElasticsearch(runCtx, tektonTaskRun)

			// VERIFY
			if test.expectedError == nil {
				assert.NilError(t, resultError)
			} else {
				assert.Error(t, resultError, test.expectedError.Error())
			}

			assert.DeepEqual(t, tektonTaskRun.Spec.Params, test.expectedParams)
		})
	}
}
func Test__runManager_Start__CreatesTektonTaskRun(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider := h.prepareMocks(mockCtrl)
	h.preparePredefinedClusterRole(mockFactory)
	config := &cfg.PipelineRunsConfigStruct{}

	examinee := NewRunManager(mockFactory, mockSecretProvider).(*runManager)
	examinee.testing = newRunManagerTestingWithRequiredStubs()

	// EXERCISE
	resultError := examinee.Start(h.ctx, mockPipelineRun, config)
	assert.NilError(t, resultError)

	// VERIFY
	runNamespace := mockPipelineRun.GetRunNamespace()
	result, err := mockFactory.TektonV1beta1().TaskRuns(runNamespace).Get(
		h.ctx, constants.TektonTaskRunName, metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Assert(t, result != nil)
}

func Test__runManager_Prepare__CleanupOnError(t *testing.T) {
	t.Parallel()

	prepareRunnamespaceErr := fmt.Errorf("cannot prepare run namespace: foo")
	cleanupError := fmt.Errorf("cannot cleanup: foo")

	for _, test := range []struct {
		name                     string
		prepareRunNamespaceError error
		cleanupError             error
		failOnCleanupCount       int
		expectedError            error
		expectedCleanupCount     int
	}{
		{
			name:                 "no failure",
			expectedCleanupCount: 1, // before, no cleanup afterwards since no error occured
		},
		{
			name:                     "failing inside prepareRunNamespace",
			prepareRunNamespaceError: prepareRunnamespaceErr,
			expectedError:            prepareRunnamespaceErr,
			expectedCleanupCount:     2, // before and after (since error occured)
		},
		{
			name:                 "failing inside initial cleanup",
			failOnCleanupCount:   1,
			cleanupError:         cleanupError,
			expectedError:        cleanupError,
			expectedCleanupCount: 1, // we are failing inside the initial cleanup, but this gets called.
		},
		{
			name:                     "failing inside defered cleanup",
			prepareRunNamespaceError: prepareRunnamespaceErr,
			failOnCleanupCount:       2,
			cleanupError:             cleanupError,
			expectedError:            prepareRunnamespaceErr, // we still expect "content" error
			expectedCleanupCount:     2,                      // we are failing inside the second (defered) cleanup
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// SETUP
			h := newTestHelper1(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockFactory, mockPipelineRun, mockSecretProvider := h.prepareMocks(mockCtrl)
			config := &cfg.PipelineRunsConfigStruct{}

			examinee := NewRunManager(mockFactory, mockSecretProvider).(*runManager)
			examinee.testing = newRunManagerTestingWithRequiredStubs()

			var cleanupCalled int
			examinee.testing.cleanupStub = func(_ context.Context, ctx *runContext) error {
				assert.Assert(t, ctx.pipelineRun == mockPipelineRun)
				cleanupCalled++
				if test.cleanupError != nil && cleanupCalled == test.failOnCleanupCount {
					return test.cleanupError
				}
				return nil
			}
			examinee.testing.prepareRunNamespaceStub = func(_ context.Context, ctx *runContext) error {
				return test.prepareRunNamespaceError
			}

			// EXERCISE
			_, _, resultError := examinee.Prepare(h.ctx, mockPipelineRun, config)

			// VERIFY
			if test.expectedError != nil {
				assert.Error(t, resultError, test.expectedError.Error())
			}
			assert.Assert(t, cleanupCalled == test.expectedCleanupCount)
		})
	}
}

func Test__runManager_addTektonTaskRunParamsForJenkinsfileRunnerImage(t *testing.T) {
	t.Parallel()

	const (
		pipelineRunsConfigDefaultImage  = "defaultImage1"
		pipelineRunsConfigDefaultPolicy = "defaultPolicy1"
	)
	examinee := runManager{}
	for _, tc := range []struct {
		name                string
		spec                *stewardv1alpha1.PipelineSpec
		expectedAddedParams tektonv1beta1.Params
	}{
		{
			name: "empty",
			spec: &stewardv1alpha1.PipelineSpec{},
			expectedAddedParams: tektonv1beta1.Params{
				tektonStringParam("JFR_IMAGE", pipelineRunsConfigDefaultImage),
				tektonStringParam("JFR_IMAGE_PULL_POLICY", pipelineRunsConfigDefaultPolicy),
			},
		},
		{
			name: "no_image_no_policy",
			spec: &stewardv1alpha1.PipelineSpec{
				JenkinsfileRunner: &stewardv1alpha1.JenkinsfileRunnerSpec{},
			},
			expectedAddedParams: tektonv1beta1.Params{
				tektonStringParam("JFR_IMAGE", pipelineRunsConfigDefaultImage),
				tektonStringParam("JFR_IMAGE_PULL_POLICY", pipelineRunsConfigDefaultPolicy),
			},
		},
		{
			name: "image_only",
			spec: &stewardv1alpha1.PipelineSpec{
				JenkinsfileRunner: &stewardv1alpha1.JenkinsfileRunnerSpec{
					Image: "foo",
				},
			},
			expectedAddedParams: tektonv1beta1.Params{
				tektonStringParam("JFR_IMAGE", "foo"),
				tektonStringParam("JFR_IMAGE_PULL_POLICY", "IfNotPresent"),
			},
		},
		{
			name: "policy_only",
			spec: &stewardv1alpha1.PipelineSpec{
				JenkinsfileRunner: &stewardv1alpha1.JenkinsfileRunnerSpec{
					ImagePullPolicy: "bar",
				},
			},
			expectedAddedParams: tektonv1beta1.Params{
				tektonStringParam("JFR_IMAGE", pipelineRunsConfigDefaultImage),
				tektonStringParam("JFR_IMAGE_PULL_POLICY", pipelineRunsConfigDefaultPolicy),
			},
		},
		{
			name: "image_and_policy",
			spec: &stewardv1alpha1.PipelineSpec{
				JenkinsfileRunner: &stewardv1alpha1.JenkinsfileRunnerSpec{
					Image:           "foo",
					ImagePullPolicy: "bar",
				},
			},
			expectedAddedParams: tektonv1beta1.Params{
				tektonStringParam("JFR_IMAGE", "foo"),
				tektonStringParam("JFR_IMAGE_PULL_POLICY", "bar"),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			t.Parallel()

			// SETUP
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockPipelineRun := k8smocks.NewMockPipelineRun(mockCtrl)
			mockPipelineRun.EXPECT().GetSpec().Return(tc.spec).AnyTimes()
			existingParam := tektonStringParam("AlreadyExistingParam1", "foo")
			tektonTaskRun := tektonv1beta1.TaskRun{
				Spec: tektonv1beta1.TaskRunSpec{
					Params: tektonv1beta1.Params{*existingParam.DeepCopy()},
				},
			}
			runCtx := &runContext{
				pipelineRun: mockPipelineRun,
				pipelineRunsConfig: &cfg.PipelineRunsConfigStruct{
					JenkinsfileRunnerImage:           pipelineRunsConfigDefaultImage,
					JenkinsfileRunnerImagePullPolicy: pipelineRunsConfigDefaultPolicy,
				},
			}

			// EXERCISE
			examinee.addTektonTaskRunParamsForJenkinsfileRunnerImage(runCtx, &tektonTaskRun)

			// VERIFY
			expectedParams := tektonv1beta1.Params{existingParam}
			expectedParams = append(expectedParams, tc.expectedAddedParams...)
			assert.DeepEqual(t, expectedParams, tektonTaskRun.Spec.Params)
		})
	}
}

func Test__runManager_Prepare__DoesNotSetPipelineRunStatus(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider := h.prepareMocks(mockCtrl)
	h.preparePredefinedClusterRole(mockFactory)
	config := &cfg.PipelineRunsConfigStruct{}

	examinee := NewRunManager(mockFactory, mockSecretProvider).(*runManager)
	examinee.testing = newRunManagerTestingWithRequiredStubs()

	// EXERCISE
	_, _, resultError := examinee.Prepare(h.ctx, mockPipelineRun, config)
	assert.NilError(t, resultError)

	// VERIFY
	// UpdateState should never be called
	mockPipelineRun.EXPECT().UpdateState(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func Test__runManager_Start__DoesNotSetPipelineRunStatus(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider := h.prepareMocks(mockCtrl)
	h.preparePredefinedClusterRole(mockFactory)
	config := &cfg.PipelineRunsConfigStruct{}

	examinee := NewRunManager(mockFactory, mockSecretProvider).(*runManager)
	examinee.testing = newRunManagerTestingWithRequiredStubs()

	// EXERCISE
	resultError := examinee.Start(h.ctx, mockPipelineRun, config)
	assert.NilError(t, resultError)

	// VERIFY
	// UpdateState should never be called
	mockPipelineRun.EXPECT().UpdateState(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func Test__runManager_copySecretsToRunNamespace__DoesCopySecret(t *testing.T) {
	t.Parallel()

	// SETUP
	ctx := context.Background()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	examinee := &runManager{}

	mockSecretManager := runmocks.NewMockSecretManager(mockCtrl)
	// inject secret manager

	examinee.testing = newRunManagerTestingWithRequiredStubs()
	examinee.testing.getSecretManagerStub = func(*runContext) runifc.SecretManager {
		return mockSecretManager
	}

	run := k8smocks.NewMockPipelineRun(mockCtrl)
	runCtx := &runContext{
		pipelineRun: run,
	}

	// EXPECT
	mockSecretManager.EXPECT().CopyAll(gomock.Not(gomock.Nil()), run).
		Return("cloneSecret1", []string{"foo", "bar"}, nil).
		Times(1)

	// EXERCISE
	cloneSecret, imagePullSecrets, resultError := examinee.copySecretsToRunNamespace(ctx, runCtx)

	// VERFIY
	assert.NilError(t, resultError)
	assert.Equal(t, "cloneSecret1", cloneSecret)
	assert.DeepEqual(t, []string{"foo", "bar"}, imagePullSecrets)
}

func Test__runManager_Cleanup__RemovesNamespaces(t *testing.T) {
	for _, ffEnabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("featureflag_CreateAuxNamespaceIfUnused_%t", ffEnabled), func(t *testing.T) {
			defer featureflagtesting.WithFeatureFlag(featureflag.CreateAuxNamespaceIfUnused, ffEnabled)()

			// SETUP
			h := newTestHelper1(t)

			cf := newFakeClientFactory(
				k8sfake.Namespace(h.namespace1),
				k8sfake.PipelineRun(h.pipelineRun1, h.namespace1, stewardv1alpha1.PipelineSpec{}),
			)

			config := &cfg.PipelineRunsConfigStruct{}
			secretProvider := secretproviderfakes.NewProvider(h.namespace1)

			examinee := NewRunManager(cf, secretProvider).(*runManager)
			examinee.testing = newRunManagerTestingWithAllNoopStubs()
			examinee.testing.cleanupStub = nil

			pipelineRunHelper, err := k8s.NewPipelineRun(h.ctx, h.getPipelineRunFromStorage(cf, h.namespace1, h.pipelineRun1), cf)
			assert.NilError(t, err)

			runCtx := &runContext{
				pipelineRun:        pipelineRunHelper,
				pipelineRunsConfig: config,
			}
			err = examinee.prepareRunNamespace(h.ctx, runCtx)
			assert.NilError(t, err)
			runCtx.pipelineRun.UpdateRunNamespace(runCtx.runNamespace)
			runCtx.pipelineRun.UpdateAuxNamespace(runCtx.auxNamespace)
			_, err = runCtx.pipelineRun.CommitStatus(h.ctx)
			assert.NilError(t, err)
			{
				pipelineRun1 := h.getPipelineRunFromStorage(cf, h.namespace1, h.pipelineRun1)
				expectedNamespaces := []string{h.namespace1}
				assert.Assert(t, pipelineRun1.Status.Namespace != "")
				expectedNamespaces = append(expectedNamespaces, pipelineRun1.Status.Namespace)
				if ffEnabled {
					assert.Assert(t, pipelineRun1.Status.AuxiliaryNamespace != "")
					expectedNamespaces = append(expectedNamespaces, pipelineRun1.Status.AuxiliaryNamespace)
				}
				h.assertThatExactlyTheseNamespacesExist(cf, expectedNamespaces...)
			}

			// EXERCISE
			resultErr := examinee.Cleanup(h.ctx, pipelineRunHelper)

			// VERIFY
			assert.NilError(t, resultErr)

			// namespaces
			{
				pipelineRun1 := h.getPipelineRunFromStorage(cf, h.namespace1, h.pipelineRun1)
				assert.Assert(t, pipelineRun1.Status.Namespace != "")
				if ffEnabled {
					assert.Assert(t, pipelineRun1.Status.AuxiliaryNamespace != "")
				}
				h.assertThatExactlyTheseNamespacesExist(cf, h.namespace1)
			}
		})
	}
}

func dummySecretProvider(factory k8s.ClientFactory, namespace string) secrets.SecretProvider {
	secretsClient := factory.CoreV1().Secrets(namespace)
	return k8ssecretprovider.NewProvider(secretsClient, namespace)
}

func Test__runManager__Log_Elasticsearch(t *testing.T) {
	t.Parallel()

	const (
		TaskRunParamNameIndexURL  = "PIPELINE_LOG_ELASTICSEARCH_INDEX_URL"
		TaskRunParamNameRunIDJSON = "PIPELINE_LOG_ELASTICSEARCH_RUN_ID_JSON"
	)

	findTaskRunParam := func(taskRun *tektonv1beta1.TaskRun, paramName string) (param *tektonv1beta1.Param) {
		assert.Assert(t, taskRun.Spec.Params != nil)
		for _, p := range taskRun.Spec.Params {
			if p.Name == paramName {
				if param != nil {
					t.Fatalf("input param specified twice: %s", paramName)
				}
				param = &p
			}
		}
		return
	}

	setupExaminee := func(
		t *testing.T, pipelineRunJSON string,
	) (
		examinee *runManager, runCtx *runContext, cf *k8sfake.ClientFactory,
	) {
		pipelineRun := runctltesting.StewardObjectFromJSON(t, pipelineRunJSON).(*stewardv1alpha1.PipelineRun)
		t.Log("decoded:\n", spew.Sdump(pipelineRun))

		cf = k8sfake.NewClientFactory(
			k8sfake.Namespace("namespace1"),
			pipelineRun,
		)
		ctx := context.Background()
		k8sPipelineRun, err := k8s.NewPipelineRun(ctx, pipelineRun, cf)
		assert.NilError(t, err)
		config := &cfg.PipelineRunsConfigStruct{}
		examinee = NewRunManager(
			cf,
			dummySecretProvider(cf, pipelineRun.GetNamespace()),
		).(*runManager)
		examinee.testing = newRunManagerTestingWithRequiredStubs()
		runCtx = &runContext{
			pipelineRun:        k8sPipelineRun,
			pipelineRunsConfig: config,
		}
		return
	}

	expectSingleTaskRun := func(t *testing.T, cf *k8sfake.ClientFactory, k8sPipelineRun k8s.PipelineRun) *tektonv1beta1.TaskRun {
		ctx := context.Background()
		taskRunList, err := cf.TektonV1beta1().TaskRuns(k8sPipelineRun.GetRunNamespace()).List(ctx, metav1.ListOptions{})
		assert.NilError(t, err)
		assert.Equal(t, 1, len(taskRunList.Items), "%s", spew.Sdump(taskRunList))
		return &taskRunList.Items[0]
	}

	/**
	 * Test: Various JSON values for spec.logging.elasticsearch.runID
	 * are correctly passed as Tekton TaskRun input parameter.
	 */
	test := "Passthrough"
	for _, tc := range []struct {
		name               string
		runIDJSON          string
		expectedParamValue string
	}{
		{"none", ``, `null`},
		{"dummy", `"___dummy___": 1`, `null`},
		{"null", `"runID": null`, `null`},
		{"true", `"runID": true`, `true`},
		{"false", `"runID": false`, `false`},
		{"int", `"runID": 123`, `123`},
		{"intneg", `"runID": -123`, `-123`},
		{"float", `"runID": 123.45`, `123.45`},
		{"floatneg", `"runID": -123.45`, `-123.45`},
		{"string", `"runID": "some string"`, `"some string"`},
		{"map", `"runID": { "key2": "value2", "key1": "value1" }`, `{"key1":"value1","key2":"value2"}`},
		{"mapdeep", `
			"runID": {
				"key1": {
					"key2": {
						"key3_1": "value3",
						"key3_2": null,
						"key3_3": [1, "2", true]
					}
				}
			}`,
			`{"key1":{"key2":{"key3_1":"value3","key3_2":null,"key3_3":[1,"2",true]}}}`},
	} {
		t.Run(test+"_"+tc.name, func(t *testing.T) {
			// setup
			ctx := context.Background()
			pipelineRunJSON := fmt.Sprintf(fixIndent(`
				{
					"apiVersion": "steward.sap.com/v1alpha1",
					"kind": "PipelineRun",
					"metadata": {
						"name": "dummy1",
						"namespace": "namespace1"
					},
					"spec": {
						"jenkinsFile": {
							"repoUrl": "dummyRepoUrl",
							"revision": "dummyRevision",
							"relativePath": "dummyRelativePath"
						},
						"logging": {
							"elasticsearch": {
								%s
							}
						}
					}
				}`),
				tc.runIDJSON,
			)
			t.Log("input:", pipelineRunJSON)
			examinee, runCtx, cf := setupExaminee(t, pipelineRunJSON)

			// exercise
			resultError := examinee.createTektonTaskRun(ctx, runCtx)
			assert.NilError(t, resultError)
			// verify
			taskRun := expectSingleTaskRun(t, cf, runCtx.pipelineRun)
			param := findTaskRunParam(taskRun, TaskRunParamNameRunIDJSON)
			assert.Assert(t, param != nil)
			assert.Equal(t, tektonv1beta1.ParamTypeString, param.Value.Type)
			assert.Equal(t, tc.expectedParamValue, param.Value.StringVal)

			param = findTaskRunParam(taskRun, TaskRunParamNameIndexURL)
			assert.Assert(t, is.Nil(param))
		})
	}

	/**
	 * Test: If there is no spec.logging.elasticsearch, the index URL
	 * template parameter should be defined as empty string, effectively
	 * disabling logging to Elasticsearch.
	 */
	test = "SuppressIndexURL"
	for _, tc := range []struct {
		name            string
		loggingFragment string
	}{
		{"NoLogging", ``},
		{"NoElasticsearch", `,"logging":{"___dummy___":1}`},
	} {
		t.Run(test+"_"+tc.name, func(t *testing.T) {
			// setup
			ctx := context.Background()
			pipelineRunJSON := fmt.Sprintf(fixIndent(`
				{
					"apiVersion": "steward.sap.com/v1alpha1",
					"kind": "PipelineRun",
					"metadata": {
						"name": "dummy1",
						"namespace": "namespace1"
					},
					"spec": {
						"jenkinsFile": {
							"repoUrl": "dummyRepoUrl",
							"revision": "dummyRevision",
							"relativePath": "dummyRelativePath"
						}
						%s
					}
				}`),
				tc.loggingFragment,
			)
			t.Log("input:", pipelineRunJSON)
			examinee, runCtx, cf := setupExaminee(t, pipelineRunJSON)

			// exercise
			resultError := examinee.createTektonTaskRun(ctx, runCtx)
			assert.NilError(t, resultError)

			// verify
			taskRun := expectSingleTaskRun(t, cf, runCtx.pipelineRun)

			param := findTaskRunParam(taskRun, TaskRunParamNameIndexURL)
			assert.Assert(t, param != nil)
			assert.Equal(t, tektonv1beta1.ParamTypeString, param.Value.Type)
			assert.Equal(t, "", param.Value.StringVal)

			param = findTaskRunParam(taskRun, TaskRunParamNameRunIDJSON)
			assert.Assert(t, is.Nil(param))
		})
	}

	/**
	 * Test: If `spec.logging.elasticsearch.indexURL` has an unsupported
	 * scheme, or format of the URL is incorrect an error is returned.
	 */
	test = "ErrorOnWrongFormattedIndexURL"
	for _, tc := range []struct {
		name          string
		URL           string
		expectedError string
	}{
		{"indexURLWithNoScheme", `"indexURL": "testURL"`, "scheme not supported"},
		{"indexURLWithIncorrectScheme", `"indexURL": "ftp://testURL"`, "scheme not supported"},
		{"indexURLWithNonsenseScheme", `"indexURL": "nonsense://testURL"`, "scheme not supported"},
		{"indexURLWithWrongFormat2", `"indexURL": "http//testURL"`, "scheme not supported"},
		{"indexURLWithWrongFormat3", `"indexURL": ":///bar"`, "missing protocol scheme"},
		{"indexURLWithWrongFormat4", `"indexURL": "http://\bar"`, "invalid control character in URL"},
		{"emptyIndexURL", `"indexURL": "http://testURL:wrongPort"`, "invalid port"},
	} {
		t.Run(test+"_"+tc.name, func(t *testing.T) {
			// setup
			ctx := context.Background()
			pipelineRunJSON := fmt.Sprintf(fixIndent(`
				{
					"apiVersion": "steward.sap.com/v1alpha1",
					"kind": "PipelineRun",
					"metadata": {
						"name": "dummy1",
						"namespace": "namespace1"
					},
					"spec": {
						"jenkinsFile": {
							"repoUrl": "dummyRepoUrl",
							"revision": "dummyRevision",
							"relativePath": "dummyRelativePath"
						},
						"logging": {
							"elasticsearch": {
								%s
							}
						}
					}
				}`),
				tc.URL,
			)
			t.Log("input:", pipelineRunJSON)
			examinee, runCtx, _ := setupExaminee(t, pipelineRunJSON)
			// exercise
			resultError := examinee.createTektonTaskRun(ctx, runCtx)
			assert.ErrorContains(t, resultError, tc.expectedError)
		})
	}

	/**
	 * Test: `createTektonTaskRun` handles valid index URLs correctly.
	 */
	test = "CorrectFormatForIndexURL"
	for _, tc := range []struct {
		name string
		URL  string
	}{
		{"validhttpURL", `"indexURL": "http://host.domain"`},
		{"validhttpsURL", `"indexURL": "https://host.domain"`},
		{"validHTTPURL", `"indexURL": "HTTP://host.domain"`},
		{"validHTTPSURL", `"indexURL": "HTTPS://host.domain"`},
	} {
		t.Run(test+"_"+tc.name, func(t *testing.T) {
			// setup
			ctx := context.Background()
			pipelineRunJSON := fmt.Sprintf(fixIndent(`
				{
					"apiVersion": "steward.sap.com/v1alpha1",
					"kind": "PipelineRun",
					"metadata": {
						"name": "dummy1",
						"namespace": "namespace1"
					},
					"spec": {
						"jenkinsFile": {
							"repoUrl": "dummyRepoUrl",
							"revision": "dummyRevision",
							"relativePath": "dummyRelativePath"
						},
						"logging": {
							"elasticsearch": {
								%s
							}
						}
					}
				}`),
				tc.URL,
			)
			t.Log("input:", pipelineRunJSON)
			examinee, runCtx, _ := setupExaminee(t, pipelineRunJSON)

			// exercise
			resultError := examinee.createTektonTaskRun(ctx, runCtx)

			// verify
			assert.NilError(t, resultError)
		})
	}
}

func Test__runManager__getTimeout__retrievesPipelineTimeoutIfSetInThePipelineSpec(t *testing.T) {
	t.Parallel()

	//SETUP
	defaultTimeout := utils.Metav1Duration(200)
	customTimeout := utils.Metav1Duration(300)
	ctx := context.Background()

	k8sPipelineRun, err := k8s.NewPipelineRun(ctx, &stewardv1alpha1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: stewardv1alpha1.SchemeGroupVersion.String(),
			Kind:       "PipelineRun",
		},
		Spec: stewardv1alpha1.PipelineSpec{Timeout: customTimeout},
	}, nil)
	assert.NilError(t, err)

	config := &cfg.PipelineRunsConfigStruct{
		Timeout: defaultTimeout,
	}
	runCtx := &runContext{
		pipelineRun:        k8sPipelineRun,
		pipelineRunsConfig: config,
	}

	//EXERCISE
	result := getTimeout(runCtx)

	//VERIFY
	assert.DeepEqual(t, customTimeout, result)

}

func Test__runManager__getTimeout__retrievesTheDefaultPipelineTimeoutIfTimeoutIsNilInThePipelineSpec(t *testing.T) {
	t.Parallel()

	//SETUP
	ctx := context.Background()
	k8sPipelineRun, err := k8s.NewPipelineRun(ctx, &stewardv1alpha1.PipelineRun{}, nil)
	assert.NilError(t, err)
	defaultTimeout := utils.Metav1Duration(200)
	config := &cfg.PipelineRunsConfigStruct{
		Timeout: defaultTimeout,
	}
	runCtx := &runContext{
		pipelineRun:        k8sPipelineRun,
		pipelineRunsConfig: config,
	}

	//EXERCISE
	result := getTimeout(runCtx)

	//VERIFY
	assert.DeepEqual(t, defaultTimeout, result)
}

func Test__runManager_GetRun_Missing(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider := h.prepareMocks(mockCtrl)
	h.addTektonTaskRun(mockFactory)

	examinee := NewRunManager(mockFactory, mockSecretProvider).(*runManager)

	// EXERCISE
	run, resultError := examinee.GetRun(h.ctx, mockPipelineRun)

	// VERIFY
	assert.NilError(t, resultError)
	assert.Assert(t, run != nil)

	mockPipelineRun.EXPECT().UpdateState(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func Test__runManager_GetRun_Existing(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider := h.prepareMocks(mockCtrl)

	examinee := NewRunManager(mockFactory, mockSecretProvider).(*runManager)

	// EXERCISE
	run, resultError := examinee.GetRun(h.ctx, mockPipelineRun)

	// VERIFY
	assert.NilError(t, resultError)
	assert.Assert(t, run == nil)

	mockPipelineRun.EXPECT().UpdateState(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func Test__runManager_DeleteRun_Success(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider := h.prepareMocks(mockCtrl)
	h.addTektonTaskRun(mockFactory)

	examinee := NewRunManager(mockFactory, mockSecretProvider).(*runManager)

	// EXERCISE
	resultError := examinee.DeleteRun(h.ctx, mockPipelineRun)

	// VERIFY
	assert.NilError(t, resultError)

	mockPipelineRun.EXPECT().UpdateState(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func Test__runManager_DeleteRun_Missing(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider := h.prepareMocks(mockCtrl)

	examinee := NewRunManager(mockFactory, mockSecretProvider)

	// EXERCISE
	resultError := examinee.DeleteRun(h.ctx, mockPipelineRun)

	// VERIFY
	assert.NilError(t, resultError)
	mockPipelineRun.EXPECT().UpdateState(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func Test__runManager_DeleteRun_MissingRunNamespace(t *testing.T) {
	t.Parallel()

	// SETUP
	h := newTestHelper1(t)
	h.runNamespace1 = ""
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider := h.prepareMocks(mockCtrl)

	examinee := NewRunManager(mockFactory, mockSecretProvider).(*runManager)

	mockPipelineRun.EXPECT().GetName().Return("foo").Times(1)

	// EXERCISE
	resultError := examinee.DeleteRun(h.ctx, mockPipelineRun)

	// VERIFY
	assert.Error(t, resultError, `cannot delete taskrun, run namespace not set in "foo"`)

	mockPipelineRun.EXPECT().UpdateState(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func Test__runManager_DeleteRun_Recoverable(t *testing.T) {
	t.Parallel()

	const (
		anyInt     = 1
		anyMessage = "message1"
	)
	var (
		anyError    = fmt.Errorf("foo")
		anyResource = resource("r1")
	)

	for _, tc := range []struct {
		name           string
		transientError error
	}{
		{"internal", k8serrors.NewInternalError(anyError)},
		{"server timeout", k8serrors.NewServerTimeout(anyResource, anyMessage, anyInt)},
		{"unavailable", k8serrors.NewServiceUnavailable(anyMessage)},
		{"timeout", k8serrors.NewTimeoutError(anyMessage, anyInt)},
		{"too many requests", k8serrors.NewTooManyRequestsError(anyMessage)},
	} {
		t.Run(tc.name, func(t *testing.T) {

			// SETUP
			h := newTestHelper1(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockFactory, mockPipelineRun, mockSecretProvider := h.prepareMocks(mockCtrl)

			h.tektonClientset.PrependReactor("delete", "*", k8sfake.NewErrorReactor(tc.transientError))
			examinee := NewRunManager(mockFactory, mockSecretProvider).(*runManager)
			// EXERCISE
			resultError := examinee.DeleteRun(h.ctx, mockPipelineRun)

			// VERIFY
			assert.Assert(t, serrors.IsRecoverable(resultError), fmt.Sprintf("%+v", resultError))

			mockPipelineRun.EXPECT().UpdateState(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		})
	}
}

func newFakeClientFactory(objects ...runtime.Object) *fake.ClientFactory {
	cf := fake.NewClientFactory(objects...)

	cf.KubernetesClientset().PrependReactor("create", "*", fake.GenerateNameReactor(0))

	cf.StewardClientset().PrependReactor("create", "*", fake.NewCreationTimestampReactor())

	return cf
}

func resource(resource string) schema.GroupResource {
	return schema.GroupResource{Group: "", Resource: resource}
}
