package runctl

import (
	"fmt"
	"strings"
	"testing"

	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	stewardfake "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/fake"
	"github.com/SAP/stewardci-core/pkg/k8s"
	fake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	k8sfake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	"github.com/SAP/stewardci-core/pkg/k8s/secrets"
	secretMocks "github.com/SAP/stewardci-core/pkg/k8s/secrets/mocks"
	tektonclientfake "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned/fake"
	"github.com/davecgh/go-spew/spew"
	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	corev1api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func newRunManagerTestingWithAllNoopStubs() *runManagerTesting {
	return &runManagerTesting{
		cleanupStub:                               func(*runContext) error { return nil },
		copySecretsToRunNamespaceStub:             func(*runContext) (string, []string, error) { return "", []string{}, nil },
		getServiceAccountSecretNameStub:           func(*runContext) string { return "" },
		setupNetworkPolicyFromConfigStub:          func(*runContext) error { return nil },
		setupLimitRangeFromConfigStub:             func(*runContext) error { return nil },
		setupResourceQuotaFromConfigStub:          func(*runContext) error { return nil },
		setupNetworkPolicyThatIsolatesAllPodsStub: func(*runContext) error { return nil },
		setupServiceAccountStub:                   func(*runContext, string, []string) error { return nil },
		setupStaticNetworkPoliciesStub:            func(*runContext) error { return nil },
		setupStaticLimitRangeStub:                 func(*runContext) error { return nil },
		setupStaticResourceQuotaStub:              func(*runContext) error { return nil },
	}
}

func newRunManagerTestingWithRequiredStubs() *runManagerTesting {
	return &runManagerTesting{
		getServiceAccountSecretNameStub: func(*runContext) string { return "" },
	}
}

func Test_RunManager_PrepareRunNamespace_Succeeds(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocks(mockCtrl)
	config := &pipelineRunsConfigStruct{}
	runCtx := &runContext{pipelineRun: mockPipelineRun}

	examinee := NewRunManager(mockFactory, config, mockSecretProvider, mockNamespaceManager).(*runManager)
	examinee.testing = newRunManagerTestingWithAllNoopStubs()

	// EXERCISE
	err := examinee.prepareRunNamespace(runCtx)

	// VERIFY
	assert.NilError(t, err)
}

func Test_RunManager_PrepareRunNamespace_CreatesNamespace(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocks(mockCtrl)
	config := &pipelineRunsConfigStruct{}
	runCtx := &runContext{pipelineRun: mockPipelineRun}

	examinee := NewRunManager(mockFactory, config, mockSecretProvider, mockNamespaceManager).(*runManager)
	examinee.testing = newRunManagerTestingWithAllNoopStubs()

	// EXERCISE
	err := examinee.prepareRunNamespace(runCtx)
	assert.NilError(t, err)

	// VERIFY
	assert.Assert(t, strings.HasPrefix(mockPipelineRun.GetRunNamespace(), runNamespacePrefix))
}

func Test_RunManager_PrepareRunNamespace_Calls_copySecretsToRunNamespace_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocks(mockCtrl)
	config := &pipelineRunsConfigStruct{}
	runCtx := &runContext{pipelineRun: mockPipelineRun}

	examinee := NewRunManager(mockFactory, config, mockSecretProvider, mockNamespaceManager).(*runManager)
	examinee.testing = newRunManagerTestingWithAllNoopStubs()

	expectedError := errors.New("some error")
	var methodCalled bool
	examinee.testing.copySecretsToRunNamespaceStub = func(ctx *runContext) (string, []string, error) {
		methodCalled = true
		assert.Assert(t, ctx.pipelineRun == mockPipelineRun)
		assert.Assert(t, ctx.runNamespace != "")
		assert.Equal(t, mockPipelineRun.GetRunNamespace(), ctx.runNamespace)
		return "", nil, expectedError
	}

	var cleanupCalled bool
	examinee.testing.cleanupStub = func(ctx *runContext) error {
		assert.Assert(t, ctx.pipelineRun == mockPipelineRun)
		cleanupCalled = true
		return nil
	}

	// EXERCISE
	resultError := examinee.prepareRunNamespace(runCtx)

	// VERIFY
	assert.Equal(t, expectedError, resultError)
	assert.Assert(t, methodCalled == true)
	assert.Assert(t, cleanupCalled == true)
}

func Test_RunManager_PrepareRunNamespace_Calls_setupServiceAccount_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocks(mockCtrl)
	config := &pipelineRunsConfigStruct{}
	runCtx := &runContext{pipelineRun: mockPipelineRun}

	examinee := NewRunManager(mockFactory, config, mockSecretProvider, mockNamespaceManager).(*runManager)
	examinee.testing = newRunManagerTestingWithAllNoopStubs()

	expectedPipelineCloneSecretName := "pipelineCloneSecret1"
	expectedImagePullSecretNames := []string{"imagePullSecret1"}
	expectedError := errors.New("some error")
	var methodCalled bool
	examinee.testing.setupServiceAccountStub = func(ctx *runContext, pipelineCloneSecretName string, imagePullSecretNames []string) error {
		methodCalled = true
		assert.Assert(t, ctx.runNamespace != "")
		assert.Equal(t, mockPipelineRun.GetRunNamespace(), ctx.runNamespace)
		assert.Equal(t, expectedPipelineCloneSecretName, pipelineCloneSecretName)
		assert.DeepEqual(t, expectedImagePullSecretNames, imagePullSecretNames)
		return expectedError
	}
	examinee.testing.copySecretsToRunNamespaceStub = func(ctx *runContext) (string, []string, error) {
		return expectedPipelineCloneSecretName, expectedImagePullSecretNames, nil
	}

	var cleanupCalled bool
	examinee.testing.cleanupStub = func(ctx *runContext) error {
		assert.Assert(t, ctx.pipelineRun == mockPipelineRun)
		cleanupCalled = true
		return nil
	}

	// EXERCISE
	resultError := examinee.prepareRunNamespace(runCtx)

	// VERIFY
	assert.Equal(t, expectedError, resultError)
	assert.Assert(t, methodCalled == true)
	assert.Assert(t, cleanupCalled == true)
}

func Test_RunManager_PrepareRunNamespace_Calls_setupStaticNetworkPolicies_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocks(mockCtrl)
	config := &pipelineRunsConfigStruct{}
	runCtx := &runContext{pipelineRun: mockPipelineRun}

	examinee := NewRunManager(mockFactory, config, mockSecretProvider, mockNamespaceManager).(*runManager)
	examinee.testing = newRunManagerTestingWithAllNoopStubs()

	expectedError := errors.New("some error")
	var methodCalled bool
	examinee.testing.setupStaticNetworkPoliciesStub = func(ctx *runContext) error {
		methodCalled = true
		assert.Assert(t, ctx.runNamespace != "")
		assert.Equal(t, mockPipelineRun.GetRunNamespace(), ctx.runNamespace)
		return expectedError
	}

	var cleanupCalled bool
	examinee.testing.cleanupStub = func(ctx *runContext) error {
		assert.Assert(t, ctx.pipelineRun == mockPipelineRun)
		cleanupCalled = true
		return nil
	}

	// EXERCISE
	resultError := examinee.prepareRunNamespace(runCtx)

	// VERIFY
	assert.Equal(t, expectedError, resultError)
	assert.Assert(t, methodCalled == true)
	assert.Assert(t, cleanupCalled == true)
}

func Test_RunManager_setupStaticNetworkPolicies_Succeeds(t *testing.T) {
	t.Parallel()

	// SETUP
	runCtx := &runContext{}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticNetworkPoliciesStub = nil

	// EXERCISE
	resultError := examinee.setupStaticNetworkPolicies(runCtx)

	// VERIFY
	assert.NilError(t, resultError)
}

func Test_RunManager_setupStaticNetworkPolicies_Calls_setupNetworkPolicyThatIsolatesAllPods_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	runNamespaceName := "runNamespace1"
	runCtx := &runContext{runNamespace: runNamespaceName}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticNetworkPoliciesStub = nil

	var methodCalled bool
	expectedError := errors.New("some error")
	examinee.testing.setupNetworkPolicyThatIsolatesAllPodsStub = func(ctx *runContext) error {
		methodCalled = true
		assert.Equal(t, runNamespaceName, ctx.runNamespace)
		return expectedError
	}

	// EXERCISE
	resultError := examinee.setupStaticNetworkPolicies(runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to set up the network policy isolating all pods in namespace \""+runNamespaceName+"\": ")
	assert.Assert(t, errors.Cause(resultError) == expectedError)
	assert.Assert(t, methodCalled == true)
}

func Test_RunManager_setupStaticNetworkPolicies_Calls_setupNetworkPolicyFromConfig_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	runNamespaceName := "runNamespace1"
	runCtx := &runContext{runNamespace: runNamespaceName}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticNetworkPoliciesStub = nil

	var methodCalled bool
	expectedError := errors.New("some error")
	examinee.testing.setupNetworkPolicyFromConfigStub = func(ctx *runContext) error {
		methodCalled = true
		assert.Equal(t, runNamespaceName, ctx.runNamespace)
		return expectedError
	}

	// EXERCISE
	resultError := examinee.setupStaticNetworkPolicies(runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to set up the configured network policy in namespace \""+runNamespaceName+"\": ")
	assert.Assert(t, errors.Cause(resultError) == expectedError)
	assert.Assert(t, methodCalled == true)
}

func Test_RunManager_setupNetworkPolicyThatIsolatesAllPods(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName   = "runNamespace1"
		expectedNamePrefix = "steward.sap.com--isolate-all-"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	cf := fake.NewClientFactory()
	cf.KubernetesClientset().PrependReactor("create", "*", fake.GenerateNameReactor(0))

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyThatIsolatesAllPodsStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyThatIsolatesAllPods(runCtx)
	assert.NilError(t, resultError)

	// VERIFY
	actualPolicies, err := cf.NetworkingV1().NetworkPolicies(runNamespaceName).List(metav1.ListOptions{})
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

func Test_RunManager_setupNetworkPolicyFromConfig_NoPolicyConfigured(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if no policy is configured.
	cf := mocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			NetworkPolicy: "", // no policy
		},
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(runCtx)
	assert.NilError(t, resultError)
}

func Test_RunManager_setupNetworkPolicyFromConfig_SetsMetadataAndLeavesOtherThingsUntouched(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName   = "runNamespace1"
		expectedNamePrefix = "steward.sap.com--configured-"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	cf := fake.NewClientFactory()
	cf.DynamicFake().PrependReactor("create", "*", fake.GenerateNameReactor(0))

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			NetworkPolicy: fixIndent(`
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
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(runCtx)
	assert.NilError(t, resultError)

	// VERIFY
	gvr := schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v123",
		Resource: "networkpolicies",
	}
	actualPolicies, err := cf.Dynamic().Resource(gvr).List(metav1.ListOptions{})
	assert.NilError(t, err)
	assert.Equal(t, 1, len(actualPolicies.Items))
	{
		policy := actualPolicies.Items[0]

		expectedMetadata := map[string]interface{}{
			"name":         expectedNamePrefix,
			"generateName": expectedNamePrefix,
			"namespace":    runNamespaceName,
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

func Test_RunManager_setupNetworkPolicyFromConfig_ReplacesAllMetadata(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName   = "runNamespace1"
		expectedNamePrefix = "steward.sap.com--configured-"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	cf := fake.NewClientFactory()
	cf.DynamicFake().PrependReactor("create", "*", fake.GenerateNameReactor(0))

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			NetworkPolicy: fixIndent(`
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
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(runCtx)
	assert.NilError(t, resultError)

	// VERIFY
	gvr := schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v123",
		Resource: "networkpolicies",
	}
	actualPolicies, err := cf.Dynamic().Resource(gvr).List(metav1.ListOptions{})
	assert.NilError(t, err)
	assert.Equal(t, 1, len(actualPolicies.Items))
	{
		policy := actualPolicies.Items[0]
		expectedMetadata := map[string]interface{}{
			"name":         expectedNamePrefix,
			"generateName": expectedNamePrefix,
			"namespace":    runNamespaceName,
			"labels": map[string]interface{}{
				stewardv1alpha1.LabelSystemManaged: "",
			},
		}
		assert.DeepEqual(t, expectedMetadata, policy.Object["metadata"])
	}
}

func Test_RunManager_setupNetworkPolicyFromConfig_MalformedPolicy(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := mocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			NetworkPolicy: ":", // malformed YAML
		},
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to decode configured network policy: ")
}

func Test_RunManager_setupNetworkPolicyFromConfig_UnexpectedGroup(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := mocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			NetworkPolicy: fixIndent(`
				apiVersion: unexpected.group/v1
				kind: NetworkPolicy
				`),
		},
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(runCtx)

	// VERIFY
	assert.Error(t, resultError,
		"configured network policy does not denote a"+
			" \"NetworkPolicy.networking.k8s.io\" but a"+
			" \"NetworkPolicy.unexpected.group\"")
}

func Test_RunManager_setupNetworkPolicyFromConfig_UnexpectedKind(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := mocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			NetworkPolicy: fixIndent(`
				apiVersion: networking.k8s.io/v1
				kind: UnexpectedKind
				`),
		},
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupNetworkPolicyFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(runCtx)

	// VERIFY
	assert.Error(t, resultError,
		"configured network policy does not denote a"+
			" \"NetworkPolicy.networking.k8s.io\" but a"+
			" \"UnexpectedKind.networking.k8s.io\"")
}

func Test_RunManager_setupStaticLimitRange_Calls_setupLimitRangeFromConfig_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	runNamespaceName := "runNamespace1"
	runCtx := &runContext{runNamespace: runNamespaceName}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticLimitRangeStub = nil

	var methodCalled bool
	expectedError := errors.New("some error")
	examinee.testing.setupLimitRangeFromConfigStub = func(ctx *runContext) error {
		methodCalled = true
		assert.Equal(t, runNamespaceName, ctx.runNamespace)
		return expectedError
	}

	// EXERCISE
	resultError := examinee.setupLimitRange(runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to set up the configured limit range in namespace \""+runNamespaceName+"\": ")
	assert.Assert(t, errors.Cause(resultError) == expectedError)
	assert.Assert(t, methodCalled == true)
}

func Test_RunManager_setupLimitRange_Succeeds(t *testing.T) {
	t.Parallel()

	// SETUP
	runCtx := &runContext{}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticLimitRangeStub = nil

	// EXERCISE
	resultError := examinee.setupLimitRange(runCtx)

	// VERIFY
	assert.NilError(t, resultError)
}

func Test_RunManager_setupLimitRangeFromConfig_NoLimitRangeConfigured(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if no policy is configured.
	cf := mocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			LimitRange: "", // no policy
		},
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupLimitRangeFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupLimitRangeFromConfig(runCtx)
	assert.NilError(t, resultError)
}

func Test_RunManager_setupLimitRangeFromConfig_MalformedLimitRange(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := mocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			LimitRange: ":", // malformed YAML
		},
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupLimitRangeFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupLimitRangeFromConfig(runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to decode configured limit range: ")
}

func Test_RunManager_setupLimitRangeFromConfig_UnexpectedGroup(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := mocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			LimitRange: fixIndent(`
                                apiVersion: unexpected.group/v1
                                kind: LimitRange
                                `),
		},
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupLimitRangeFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupLimitRangeFromConfig(runCtx)

	// VERIFY
	assert.Error(t, resultError,
		"configured limit range does not denote a"+
			" \"LimitRange\" but a"+
			" \"LimitRange.unexpected.group\"")
}

func Test_RunManager_setupLimitRangeFromConfig_UnexpectedKind(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := mocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			LimitRange: fixIndent(`
                                apiVersion: v1
                                kind: UnexpectedKind
                                `),
		},
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupLimitRangeFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupLimitRangeFromConfig(runCtx)

	// VERIFY
	assert.Error(t, resultError,
		"configured limit range does not denote a"+
			" \"LimitRange\" but a"+
			" \"UnexpectedKind\"")
}

func Test_RunManager_setupStaticResourceQuota_Calls_setupResourceQuotaFromConfig_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	runNamespaceName := "runNamespace1"
	runCtx := &runContext{runNamespace: runNamespaceName}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticResourceQuotaStub = nil

	var methodCalled bool
	expectedError := errors.New("some error")
	examinee.testing.setupResourceQuotaFromConfigStub = func(ctx *runContext) error {
		methodCalled = true
		assert.Equal(t, runNamespaceName, ctx.runNamespace)
		return expectedError
	}

	// EXERCISE
	resultError := examinee.setupResourceQuota(runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to set up the configured resource quota in namespace \""+runNamespaceName+"\": ")
	assert.Assert(t, errors.Cause(resultError) == expectedError)
	assert.Assert(t, methodCalled == true)
}

func Test_RunManager_setupResourceQuota_Succeeds(t *testing.T) {
	t.Parallel()

	// SETUP
	runCtx := &runContext{}
	examinee := runManager{
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupStaticResourceQuotaStub = nil

	// EXERCISE
	resultError := examinee.setupResourceQuota(runCtx)

	// VERIFY
	assert.NilError(t, resultError)
}

func Test_RunManager_setupResourceQuotaFromConfig_NoQuotaConfigured(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if no policy is configured.
	cf := mocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			LimitRange: "", // no policy
		},
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupResourceQuotaFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupResourceQuotaFromConfig(runCtx)
	assert.NilError(t, resultError)
}

func Test_RunManager_setupResourceQuotaFromConfig_MalformedResourceQuota(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := mocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			ResourceQuota: ":", // malformed YAML
		},
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupResourceQuotaFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupResourceQuotaFromConfig(runCtx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to decode configured resource quota: ")
}

func Test_RunManager_setupResourceQuotaFromConfig_UnexpectedGroup(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := mocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			ResourceQuota: fixIndent(`
                                apiVersion: unexpected.group/v1
                                kind: ResourceQuota
                                `),
		},
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupResourceQuotaFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupResourceQuotaFromConfig(runCtx)

	// VERIFY
	assert.Error(t, resultError,
		"configured resource quota does not denote a"+
			" \"ResourceQuota\" but a"+
			" \"ResourceQuota.unexpected.group\"")
}

func Test_RunManager_setupResourceQuotaFromConfig_UnexpectedKind(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	runCtx := &runContext{runNamespace: runNamespaceName}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := mocks.NewMockClientFactory(mockCtrl)

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			ResourceQuota: fixIndent(`
                                apiVersion: v1
                                kind: UnexpectedKind
                                `),
		},
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.setupResourceQuotaFromConfigStub = nil

	// EXERCISE
	resultError := examinee.setupResourceQuotaFromConfig(runCtx)

	// VERIFY
	assert.Error(t, resultError,
		"configured resource quota does not denote a"+
			" \"ResourceQuota\" but a"+
			" \"UnexpectedKind\"")
}

func Test_RunManager_createTektonTaskRun_PodTemplate_IsNotEmptyIfNoValuesToSet(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	_, mockPipelineRun, _, _ := prepareMocks(mockCtrl)
	runCtx := &runContext{
		pipelineRun:  mockPipelineRun,
		runNamespace: runNamespaceName,
	}
	mockPipelineRun.UpdateRunNamespace(runNamespaceName)
	cf := fake.NewClientFactory()

	examinee := runManager{
		factory: cf,
		testing: newRunManagerTestingWithAllNoopStubs(),
	}

	// EXERCISE
	resultError := examinee.createTektonTaskRun(runCtx)

	// VERIFY
	assert.NilError(t, resultError)

	taskRun, err := cf.TektonV1alpha1().TaskRuns(runNamespaceName).Get(tektonClusterTaskName, metav1.GetOptions{})
	assert.NilError(t, err)
	if equality.Semantic.DeepEqual(taskRun.Spec.PodTemplate, tekton.PodTemplate{}) {
		t.Fatal("podTemplate of TaskRun is empty")
	}
}

func Test_RunManager_createTektonTaskRun_PodTemplate_AllValuesSet(t *testing.T) {
	t.Parallel()

	int32Ptr := func(val int32) *int32 { return &val }
	int64Ptr := func(val int64) *int64 { return &val }

	// SETUP
	const (
		runNamespaceName         = "runNamespace1"
		serviceAccountSecretName = "foo"
	)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	_, mockPipelineRun, _, _ := prepareMocks(mockCtrl)
	mockPipelineRun.UpdateRunNamespace(runNamespaceName)
	runCtx := &runContext{
		pipelineRun:  mockPipelineRun,
		runNamespace: runNamespaceName,
	}
	cf := fake.NewClientFactory()

	examinee := runManager{
		factory: cf,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			JenkinsfileRunnerPodSecurityContextFSGroup:    int64Ptr(1111),
			JenkinsfileRunnerPodSecurityContextRunAsGroup: int64Ptr(2222),
			JenkinsfileRunnerPodSecurityContextRunAsUser:  int64Ptr(3333),
		},
		testing: newRunManagerTestingWithAllNoopStubs(),
	}
	examinee.testing.getServiceAccountSecretNameStub = func(ctx *runContext) string {
		return serviceAccountSecretName
	}

	// EXERCISE
	resultError := examinee.createTektonTaskRun(runCtx)

	// VERIFY
	assert.NilError(t, resultError)

	taskRun, err := cf.TektonV1alpha1().TaskRuns(runNamespaceName).Get(tektonClusterTaskName, metav1.GetOptions{})
	assert.NilError(t, err)
	expectedPodTemplate := tekton.PodTemplate{
		SecurityContext: &corev1api.PodSecurityContext{
			FSGroup:    int64Ptr(1111),
			RunAsGroup: int64Ptr(2222),
			RunAsUser:  int64Ptr(3333),
		},
		Volumes: []corev1api.Volume{
			{
				Name: "service-account-token",
				VolumeSource: corev1api.VolumeSource{
					Secret: &corev1api.SecretVolumeSource{
						SecretName:  serviceAccountSecretName,
						DefaultMode: int32Ptr(0644),
					},
				},
			},
		},
	}
	podTemplate := taskRun.Spec.PodTemplate
	assert.DeepEqual(t, expectedPodTemplate, podTemplate)
	assert.Assert(t, podTemplate.SecurityContext.FSGroup != examinee.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextFSGroup)
	assert.Assert(t, podTemplate.SecurityContext.RunAsGroup != examinee.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextRunAsGroup)
	assert.Assert(t, podTemplate.SecurityContext.RunAsUser != examinee.pipelineRunsConfig.JenkinsfileRunnerPodSecurityContextRunAsUser)
}

func Test_RunManager_Start_CreatesTektonTaskRun(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocks(mockCtrl)
	preparePredefinedClusterRole(t, mockFactory, mockPipelineRun)
	config := &pipelineRunsConfigStruct{}

	examinee := NewRunManager(mockFactory, config, mockSecretProvider, mockNamespaceManager).(*runManager)
	examinee.testing = newRunManagerTestingWithRequiredStubs()

	// EXERCISE
	err := examinee.Start(mockPipelineRun)
	assert.NilError(t, err)

	// VERIFY
	result, err := mockFactory.TektonV1alpha1().TaskRuns(mockPipelineRun.GetRunNamespace()).Get(
		tektonTaskRunName, metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Assert(t, result != nil)
}

func Test_RunManager_Start_DoesNotSetPipelineRunStatus(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocks(mockCtrl)
	preparePredefinedClusterRole(t, mockFactory, mockPipelineRun)
	config := &pipelineRunsConfigStruct{}

	examinee := NewRunManager(mockFactory, config, mockSecretProvider, mockNamespaceManager).(*runManager)
	examinee.testing = newRunManagerTestingWithRequiredStubs()

	// EXERCISE
	err := examinee.Start(mockPipelineRun)
	assert.NilError(t, err)

	// VERIFY
	// UpdateState should never be called
	mockPipelineRun.EXPECT().UpdateState(gomock.Any()).Times(0)
}

func Test_RunManager_Start_DoesCopySecret(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	spec := &stewardv1alpha1.PipelineSpec{
		JenkinsFile: stewardv1alpha1.JenkinsFile{
			RepoAuthSecret: "scm_secret1",
		},
		Secrets: []string{
			"secret1",
			"secret2",
		},
		ImagePullSecrets: []string{
			"imagePullSecret1",
			"imagePullSecret2",
		},
	}
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocksWithSpec(mockCtrl, spec)
	// UpdateState should never be called
	mockPipelineRun.EXPECT().
		UpdateState(gomock.Any()).
		Do(func(interface{}) { panic("unexpected call") }).
		AnyTimes()

	preparePredefinedClusterRole(t, mockFactory, mockPipelineRun)
	config := &pipelineRunsConfigStruct{}
	examinee := NewRunManager(mockFactory, config, mockSecretProvider, mockNamespaceManager).(*runManager)
	mockSecretHelper := secretMocks.NewMockSecretHelper(mockCtrl)

	// inject secret helper mock
	examinee.testing = newRunManagerTestingWithRequiredStubs()
	examinee.testing.getSecretHelperStub = func(string, corev1.SecretInterface) secrets.SecretHelper {
		return mockSecretHelper
	}

	// EXPECT
	mockSecretHelper.EXPECT().
		CopySecrets([]string{"scm_secret1"}, nil, gomock.Any()).
		Return([]string{"scm_secret1"}, nil)
	mockSecretHelper.EXPECT().
		CopySecrets([]string{"secret1", "secret2"}, nil, gomock.Any()).
		Return([]string{"secret1", "secret2"}, nil)
	mockSecretHelper.EXPECT().
		CopySecrets([]string{"imagePullSecret1", "imagePullSecret2"}, gomock.Any(), gomock.Any()).
		Return([]string{"imagePullSecret1", "imagePullSecret2"}, nil)

	// EXERCISE
	err := examinee.Start(mockPipelineRun)
	assert.NilError(t, err)
}

func Test_RunManager_Start_FailsWithContentErrorWhenPipelineCloneSecretNotFound(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	secretName := "secret1"
	spec := &stewardv1alpha1.PipelineSpec{
		JenkinsFile: stewardv1alpha1.JenkinsFile{
			RepoAuthSecret: secretName,
		},
	}
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocksWithSpec(mockCtrl, spec)

	preparePredefinedClusterRole(t, mockFactory, mockPipelineRun)
	config := &pipelineRunsConfigStruct{}
	examinee := NewRunManager(mockFactory, config, mockSecretProvider, mockNamespaceManager).(*runManager)
	examinee.testing = newRunManagerTestingWithRequiredStubs()

	// EXPECT
	mockSecretProvider.EXPECT().GetSecret(secretName).Return(nil, nil)
	mockPipelineRun.EXPECT().UpdateMessage(secrets.NewNotFoundError(secretName).Error())
	mockPipelineRun.EXPECT().UpdateResult(stewardv1alpha1.ResultErrorContent)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	err := examinee.Start(mockPipelineRun)
	assert.Assert(t, err != nil)
	assert.Assert(t, is.Regexp("failed to copy pipeline clone secret: .*", err.Error()))
}

func Test_RunManager_Start_FailsWithContentErrorWhenSecretNotFound(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	secretName := "secret1"
	spec := &stewardv1alpha1.PipelineSpec{
		Secrets: []string{secretName},
	}
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocksWithSpec(mockCtrl, spec)

	preparePredefinedClusterRole(t, mockFactory, mockPipelineRun)
	config := &pipelineRunsConfigStruct{}
	examinee := NewRunManager(mockFactory, config, mockSecretProvider, mockNamespaceManager).(*runManager)
	examinee.testing = newRunManagerTestingWithRequiredStubs()

	// EXPECT
	mockSecretProvider.EXPECT().GetSecret(secretName).Return(nil, nil)
	mockPipelineRun.EXPECT().UpdateMessage(secrets.NewNotFoundError(secretName).Error())
	mockPipelineRun.EXPECT().UpdateResult(stewardv1alpha1.ResultErrorContent)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	err := examinee.Start(mockPipelineRun)
	assert.Assert(t, err != nil)
	assert.Assert(t, is.Regexp("failed to copy pipeline secrets: .*", err.Error()))
}

func Test_RunManager_Start_FailsWithContentErrorWhenImagePullSecretNotFound(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	secretName := "secret1"
	spec := &stewardv1alpha1.PipelineSpec{
		ImagePullSecrets: []string{secretName},
	}
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocksWithSpec(mockCtrl, spec)

	preparePredefinedClusterRole(t, mockFactory, mockPipelineRun)
	config := &pipelineRunsConfigStruct{}
	examinee := NewRunManager(mockFactory, config, mockSecretProvider, mockNamespaceManager).(*runManager)
	examinee.testing = newRunManagerTestingWithRequiredStubs()

	// EXPECT
	mockSecretProvider.EXPECT().GetSecret(secretName).Return(nil, nil)
	mockPipelineRun.EXPECT().UpdateMessage(secrets.NewNotFoundError(secretName).Error())
	mockPipelineRun.EXPECT().UpdateResult(stewardv1alpha1.ResultErrorContent)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	err := examinee.Start(mockPipelineRun)
	assert.Assert(t, err != nil)
	assert.Assert(t, is.Regexp("failed to copy image pull secrets: .*", err.Error()))
}

func Test_RunManager_Start_FailsWithInfraErrorWhenForbidden(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	secretName := "scm_secret1"
	spec := &stewardv1alpha1.PipelineSpec{
		JenkinsFile: stewardv1alpha1.JenkinsFile{
			RepoAuthSecret: secretName,
		},
	}
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocksWithSpec(mockCtrl, spec)

	preparePredefinedClusterRole(t, mockFactory, mockPipelineRun)
	config := &pipelineRunsConfigStruct{}
	examinee := NewRunManager(mockFactory, config, mockSecretProvider, mockNamespaceManager).(*runManager)
	examinee.testing = newRunManagerTestingWithRequiredStubs()

	// EXPECT
	mockSecretProvider.EXPECT().GetSecret(secretName).Return(nil, fmt.Errorf("Forbidden"))
	mockPipelineRun.EXPECT().UpdateMessage("Forbidden")
	mockPipelineRun.EXPECT().UpdateResult(stewardv1alpha1.ResultErrorInfra)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	err := examinee.Start(mockPipelineRun)
	assert.Assert(t, err != nil)
}

func Test_RunManager_Cleanup_RemovesNamespace(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockFactory, mockPipelineRun, mockSecretProvider, mockNamespaceManager := prepareMocks(mockCtrl)
	preparePredefinedClusterRole(t, mockFactory, mockPipelineRun)

	config := &pipelineRunsConfigStruct{}
	examinee := NewRunManager(mockFactory, config, mockSecretProvider, mockNamespaceManager).(*runManager)
	examinee.testing = newRunManagerTestingWithRequiredStubs()
	err := examinee.prepareRunNamespace(&runContext{pipelineRun: mockPipelineRun})
	assert.NilError(t, err)
	//TODO: mockNamespaceManager.EXPECT().Create()...

	// EXERCISE
	examinee.Cleanup(mockPipelineRun)
	//TODO: mockNamespaceManager.EXPECT().Delete()...
}

func Test_RunManager_Log_Elasticsearch(t *testing.T) {
	t.Parallel()

	const (
		TaskRunParamNameIndexURL  = "PIPELINE_LOG_ELASTICSEARCH_INDEX_URL"
		TaskRunParamNameRunIDJSON = "PIPELINE_LOG_ELASTICSEARCH_RUN_ID_JSON"
	)

	findTaskRunParam := func(taskRun *tekton.TaskRun, paramName string) (param *tekton.Param) {
		assert.Assert(t, taskRun.Spec.Inputs.Params != nil)
		for _, p := range taskRun.Spec.Inputs.Params {
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
		pipelineRun := StewardObjectFromJSON(t, pipelineRunJSON).(*stewardv1alpha1.PipelineRun)
		t.Log("decoded:\n", spew.Sdump(pipelineRun))

		cf = k8sfake.NewClientFactory(
			k8sfake.Namespace("namespace1"),
			pipelineRun,
		)
		k8sPipelineRun, err := k8s.NewPipelineRun(pipelineRun, cf)
		assert.NilError(t, err)
		config := &pipelineRunsConfigStruct{}
		examinee = NewRunManager(
			cf,
			config,
			k8s.NewTenantNamespace(cf, pipelineRun.GetNamespace()).GetSecretProvider(),
			k8s.NewNamespaceManager(cf, "prefix1", 0),
		).(*runManager)
		examinee.testing = newRunManagerTestingWithRequiredStubs()
		runCtx = &runContext{
			pipelineRun: k8sPipelineRun,
		}
		return
	}

	expectSingleTaskRun := func(t *testing.T, cf *k8sfake.ClientFactory, k8sPipelineRun k8s.PipelineRun) *tekton.TaskRun {
		taskRunList, err := cf.TektonV1alpha1().TaskRuns(k8sPipelineRun.GetRunNamespace()).List(metav1.ListOptions{})
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
		{"none2", `"___dummy___": 1`, `null`},
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
			var err error

			// setup
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
			err = examinee.createTektonTaskRun(runCtx)
			assert.NilError(t, err)

			// verify
			taskRun := expectSingleTaskRun(t, cf, runCtx.pipelineRun)

			param := findTaskRunParam(taskRun, TaskRunParamNameRunIDJSON)
			assert.Assert(t, param != nil)
			assert.Equal(t, tekton.ParamTypeString, param.Value.Type)
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
			var err error

			// setup
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
			err = examinee.createTektonTaskRun(runCtx)
			assert.NilError(t, err)

			// verify
			taskRun := expectSingleTaskRun(t, cf, runCtx.pipelineRun)

			param := findTaskRunParam(taskRun, TaskRunParamNameIndexURL)
			assert.Assert(t, param != nil)
			assert.Equal(t, tekton.ParamTypeString, param.Value.Type)
			assert.Equal(t, "", param.Value.StringVal)

			param = findTaskRunParam(taskRun, TaskRunParamNameRunIDJSON)
			assert.Assert(t, is.Nil(param))
		})
	}
}

func preparePredefinedClusterRole(t *testing.T, factory *mocks.MockClientFactory, pipelineRun *mocks.MockPipelineRun) {
	// Create expected cluster role
	_, err := factory.RbacV1beta1().ClusterRoles().Create(k8sfake.ClusterRole(string(runClusterRoleName)))
	assert.NilError(t, err)
}

func prepareMocks(ctrl *gomock.Controller) (*mocks.MockClientFactory, *mocks.MockPipelineRun, *secretMocks.MockSecretProvider, k8s.NamespaceManager) {
	return prepareMocksWithSpec(ctrl, &stewardv1alpha1.PipelineSpec{})
}

func prepareMocksWithSpec(ctrl *gomock.Controller, spec *stewardv1alpha1.PipelineSpec) (*mocks.MockClientFactory, *mocks.MockPipelineRun, *secretMocks.MockSecretProvider, k8s.NamespaceManager) {
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

	runNamespace := ""
	mockPipelineRun := mocks.NewMockPipelineRun(ctrl)
	mockPipelineRun.EXPECT().GetSpec().Return(spec).AnyTimes()
	mockPipelineRun.EXPECT().GetStatus().Return(&stewardv1alpha1.PipelineStatus{}).AnyTimes()
	mockPipelineRun.EXPECT().GetKey().Return("key").AnyTimes()
	mockPipelineRun.EXPECT().GetPipelineRepoServerURL().Return("server", nil).AnyTimes()
	mockPipelineRun.EXPECT().GetRunNamespace().DoAndReturn(func() string {
		return runNamespace
	}).AnyTimes()

	mockPipelineRun.EXPECT().UpdateRunNamespace(gomock.Any()).Do(func(arg string) {
		runNamespace = arg
	}).MaxTimes(1)

	mockSecretProvider := secretMocks.NewMockSecretProvider(ctrl)

	//TODO: Mock when required
	namespaceManager := k8s.NewNamespaceManager(mockFactory, runNamespacePrefix, runNamespaceRandomLength)

	return mockFactory, mockPipelineRun, mockSecretProvider, namespaceManager
}
