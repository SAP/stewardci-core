// +build todo

package runctl

import (
	"context"
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

func newRunManagerTestingWithAllNoopStubs() *runInstanceTesting {
	return &runInstanceTesting{
		cleanupStub:                               func(context.Context) error { return nil },
		copySecretsToRunNamespaceStub:             func(context.Context) (string, []string, error) { return "", []string{}, nil },
		setupNetworkPolicyFromConfigStub:          func(context.Context) error { return nil },
		setupNetworkPolicyThatIsolatesAllPodsStub: func(context.Context) error { return nil },
		setupServiceAccountStub:                   func(context.Context, string, []string) error { return nil },
		setupStaticNetworkPoliciesStub:            func(context.Context) error { return nil },
		getServiceAccountSecretNameStub:           func(context.Context) string { return "" },
	}
}

func newRunManagerTestingWithRequiredStubs() *runInstanceTesting {
	return &runInstanceTesting{
		getServiceAccountSecretNameStub: func(context.Context) string { return "foo" },
	}
}

func Test_RunManager_PrepareRunNamespace_Succeeds(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := mockContext(mockCtrl)
	examinee := prepareMocks(mockCtrl)

	// EXERCISE
	err := examinee.prepareRunNamespace(ctx)

	// VERIFY
	assert.NilError(t, err)
}

func Test_RunManager_PrepareRunNamespace_CreatesNamespace(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := mockContext(mockCtrl)
	examinee := prepareMocks(mockCtrl)

	// EXERCISE
	err := examinee.prepareRunNamespace(ctx)
	assert.NilError(t, err)

	// VERIFY
	assert.Assert(t, strings.HasPrefix(examinee.pipelineRun.GetRunNamespace(), runNamespacePrefix))
}

func Test_RunManager_PrepareRunNamespace_Calls_copySecretsToRunNamespace_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := mockContext(mockCtrl)
	examinee := prepareMocks(mockCtrl)

	expectedError := errors.New("some error")
	var methodCalled bool
	testing := GetRunInstanceTesting(ctx)
	testing.copySecretsToRunNamespaceStub = func(ctx context.Context) (string, []string, error) {
		methodCalled = true
		return "", nil, expectedError
	}
	var cleanupCalled bool

	testing.cleanupStub = func(ctx context.Context) error {
		cleanupCalled = true
		return nil
	}

	ctx = WithRunInstanceTesting(ctx, testing)

	// EXERCISE
	resultError := examinee.prepareRunNamespace(ctx)

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
	ctx := mockContext(mockCtrl)
	examinee := prepareMocks(mockCtrl)
	expectedPipelineCloneSecretName := "pipelineCloneSecret1"
	expectedImagePullSecretNames := []string{"imagePullSecret1"}
	expectedError := errors.New("some error")
	var methodCalled bool
	testing := newRunManagerTestingWithAllNoopStubs()

	testing.setupServiceAccountStub = func(ctx context.Context, pipelineCloneSecretName string, imagePullSecretNames []string) error {
		methodCalled = true
		assert.Equal(t, expectedPipelineCloneSecretName, pipelineCloneSecretName)
		assert.DeepEqual(t, expectedImagePullSecretNames, imagePullSecretNames)
		return expectedError
	}
	testing.copySecretsToRunNamespaceStub = func(ctx context.Context) (string, []string, error) {
		return expectedPipelineCloneSecretName, expectedImagePullSecretNames, nil
	}

	var cleanupCalled bool
	testing.cleanupStub = func(ctx context.Context) error {
		cleanupCalled = true
		return nil
	}
	ctx = WithRunInstanceTesting(ctx, testing)
	// EXERCISE
	resultError := examinee.prepareRunNamespace(ctx)

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
	ctx := mockContext(mockCtrl)
	examinee := prepareMocks(mockCtrl)

	expectedError := errors.New("some error")
	var methodCalled bool
	testing := newRunManagerTestingWithAllNoopStubs()
	testing.setupStaticNetworkPoliciesStub = func(ctx context.Context) error {
		methodCalled = true
		return expectedError
	}

	var cleanupCalled bool
	testing.cleanupStub = func(ctx context.Context) error {
		cleanupCalled = true
		return nil
	}
	ctx = WithRunInstanceTesting(ctx, testing)
	// EXERCISE
	resultError := examinee.prepareRunNamespace(ctx)

	// VERIFY
	assert.Equal(t, expectedError, resultError)
	assert.Assert(t, methodCalled == true)
	assert.Assert(t, cleanupCalled == true)
}

func Test_RunManager_setupStaticNetworkPolicies_Succeeds(t *testing.T) {
	t.Parallel()

	// SETUP
	testing := newRunManagerTestingWithAllNoopStubs()
	testing.setupStaticNetworkPoliciesStub = nil
	ctx := WithRunInstanceTesting(context.TODO(), testing)
	examinee := &runInstance{}
	// EXERCISE
	resultError := examinee.setupStaticNetworkPolicies(ctx)

	// VERIFY
	assert.NilError(t, resultError)
}

func Test_RunManager_setupStaticNetworkPolicies_Calls_setupNetworkPolicyThatIsolatesAllPods_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	runNamespaceName := "runNamespace1"
	examinee := &runInstance{
		runNamespace: runNamespaceName,
	}
	testing := newRunManagerTestingWithAllNoopStubs()
	testing.setupStaticNetworkPoliciesStub = nil

	var methodCalled bool
	expectedError := errors.New("some error")
	testing.setupNetworkPolicyThatIsolatesAllPodsStub = func(ctx context.Context) error {
		methodCalled = true
		return expectedError
	}
	ctx := WithRunInstanceTesting(context.TODO(), testing)

	// EXERCISE
	resultError := examinee.setupStaticNetworkPolicies(ctx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to set up the network policy isolating all pods in namespace \""+runNamespaceName+"\": ")
	assert.Assert(t, errors.Cause(resultError) == expectedError)
	assert.Assert(t, methodCalled == true)
}

func Test_RunManager_setupStaticNetworkPolicies_Calls_setupNetworkPolicyFromConfig_AndPropagatesError(t *testing.T) {
	t.Parallel()

	// SETUP
	runNamespaceName := "runNamespace1"
	examinee := &runInstance{
		runNamespace: runNamespaceName,
	}
	testing := newRunManagerTestingWithAllNoopStubs()
	testing.setupStaticNetworkPoliciesStub = nil

	var methodCalled bool
	expectedError := errors.New("some error")
	testing.setupNetworkPolicyFromConfigStub = func(ctx context.Context) error {
		methodCalled = true
		return expectedError
	}
	ctx := WithRunInstanceTesting(context.TODO(), testing)
	// EXERCISE
	resultError := examinee.setupStaticNetworkPolicies(ctx)

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
	examinee := &runInstance{runNamespace: runNamespaceName}
	cf := fake.NewClientFactory()
	cf.KubernetesClientset().PrependReactor("create", "*", fake.GenerateNameReactor(0))
	ctx := k8s.WithClientFactory(context.TODO(), cf)
	testing := newRunManagerTestingWithAllNoopStubs()
	testing.setupNetworkPolicyThatIsolatesAllPodsStub = nil
	ctx = WithRunInstanceTesting(ctx, testing)
	// EXERCISE
	resultError := examinee.setupNetworkPolicyThatIsolatesAllPods(ctx)
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
	examinee := &runInstance{runNamespace: runNamespaceName,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			NetworkPolicy: "", // no policy
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if no policy is configured.
	cf := mocks.NewMockClientFactory(mockCtrl)
	ctx := k8s.WithClientFactory(context.TODO(), cf)
	testing := newRunManagerTestingWithAllNoopStubs()
	testing.setupNetworkPolicyFromConfigStub = nil
	ctx = WithRunInstanceTesting(ctx, testing)

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(ctx)
	assert.NilError(t, resultError)
}

func Test_RunManager_setupNetworkPolicyFromConfig_SetsMetadataAndLeavesOtherThingsUntouched(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName   = "runNamespace1"
		expectedNamePrefix = "steward.sap.com--configured-"
	)
	cf := fake.NewClientFactory()
	cf.DynamicFake().PrependReactor("create", "*", fake.GenerateNameReactor(0))
	ctx := k8s.WithClientFactory(context.TODO(), cf)
	examinee := &runInstance{
		runNamespace: runNamespaceName,
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
	}
	testing := newRunManagerTestingWithAllNoopStubs()
	testing.setupNetworkPolicyFromConfigStub = nil
	ctx = WithRunInstanceTesting(ctx, testing)

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(ctx)
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
	cf := fake.NewClientFactory()
	cf.DynamicFake().PrependReactor("create", "*", fake.GenerateNameReactor(0))
	ctx := k8s.WithClientFactory(context.TODO(), cf)
	examinee := &runInstance{
		runNamespace: runNamespaceName,
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
	}
	testing := newRunManagerTestingWithAllNoopStubs()
	testing.setupNetworkPolicyFromConfigStub = nil
	ctx = WithRunInstanceTesting(ctx, testing)

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(ctx)
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
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := mocks.NewMockClientFactory(mockCtrl)
	ctx := k8s.WithClientFactory(context.TODO(), cf)
	examinee := &runInstance{
		runNamespace: runNamespaceName,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			NetworkPolicy: ":", // malformed YAML
		},
	}
	testing := newRunManagerTestingWithAllNoopStubs()
	testing.setupNetworkPolicyFromConfigStub = nil
	ctx = WithRunInstanceTesting(ctx, testing)

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(ctx)

	// VERIFY
	assert.ErrorContains(t, resultError, "failed to decode configured network policy: ")
}

func Test_RunManager_setupNetworkPolicyFromConfig_UnexpectedGroup(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := mocks.NewMockClientFactory(mockCtrl)
	ctx := k8s.WithClientFactory(context.TODO(), cf)
	examinee := &runInstance{
		runNamespace: runNamespaceName,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			NetworkPolicy: fixIndent(`
				apiVersion: unexpected.group/v1
				kind: NetworkPolicy
				`),
		},
	}
	testing := newRunManagerTestingWithAllNoopStubs()
	testing.setupNetworkPolicyFromConfigStub = nil
	ctx = WithRunInstanceTesting(ctx, testing)

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(ctx)

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
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We use a mocked client factory without expected calls, because
	// the SUT should not use it if policy decoding fails.
	cf := mocks.NewMockClientFactory(mockCtrl)
	ctx := k8s.WithClientFactory(context.TODO(), cf)

	examinee := &runInstance{
		runNamespace: runNamespaceName,
		pipelineRunsConfig: pipelineRunsConfigStruct{
			NetworkPolicy: fixIndent(`
				apiVersion: networking.k8s.io/v1
				kind: UnexpectedKind
				`),
		},
	}
	testing := newRunManagerTestingWithAllNoopStubs()
	testing.setupNetworkPolicyFromConfigStub = nil
	ctx = WithRunInstanceTesting(ctx, testing)

	// EXERCISE
	resultError := examinee.setupNetworkPolicyFromConfig(ctx)

	// VERIFY
	assert.Error(t, resultError,
		"configured network policy does not denote a"+
			" \"NetworkPolicy.networking.k8s.io\" but a"+
			" \"UnexpectedKind.networking.k8s.io\"")
}

func Test_RunManager_createTektonTaskRun_PodTemplate_IsNotEmptyIfNoValuesToSet(t *testing.T) {
	t.Parallel()

	// SETUP
	const (
		runNamespaceName = "runNamespace1"
	)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	examinee := prepareMocks(mockCtrl)
	examinee.runNamespace = runNamespaceName
	examinee.pipelineRun.UpdateRunNamespace(runNamespaceName)
	cf := fake.NewClientFactory()
	ctx := k8s.WithClientFactory(context.TODO(), cf)
	ctx = WithRunInstanceTesting(ctx, newRunManagerTestingWithAllNoopStubs())
	// EXERCISE
	resultError := examinee.createTektonTaskRun(ctx)

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
	examinee := prepareMocks(mockCtrl)
	examinee.pipelineRun.UpdateRunNamespace(runNamespaceName)
	examinee.runNamespace = runNamespaceName
	examinee.pipelineRunsConfig = pipelineRunsConfigStruct{
		JenkinsfileRunnerPodSecurityContextFSGroup:    int64Ptr(1111),
		JenkinsfileRunnerPodSecurityContextRunAsGroup: int64Ptr(2222),
		JenkinsfileRunnerPodSecurityContextRunAsUser:  int64Ptr(3333),
	}
	cf := fake.NewClientFactory()
	ctx := k8s.WithClientFactory(context.TODO(), cf)
	ctx = WithRunInstanceTesting(ctx, newRunManagerTestingWithRequiredStubs())
	// EXERCISE
	resultError := examinee.createTektonTaskRun(ctx)

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

func Test_RunManager_StartPipelineRun_CreatesTektonTaskRun(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := mockContext(mockCtrl)
	pipelineRun := prepareMocksWithSpec(mockCtrl, nil)
	preparePredefinedClusterRole(t, ctx)
	ctx = WithRunInstanceTesting(ctx, newRunManagerTestingWithAllNoopStubs())
	config := &pipelineRunsConfigStruct{}

	// EXERCISE
	err := StartPipelineRun(ctx, examinee.pipelineRun, config)
	assert.NilError(t, err)

	// VERIFY
	result, err := k8s.GetClientFactory(ctx).TektonV1alpha1().TaskRuns(examinee.pipelineRun.GetRunNamespace()).Get(
		tektonTaskRunName, metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Assert(t, result != nil)
}

func Test_RunManager_StartPipelineRun_DoesNotSetPipelineRunStatus(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := mockContext(mockCtrl)
	mockPipelineRun := prepareMocksWithSpec(mockCtrl, nil)
	preparePredefinedClusterRole(t, ctx)
	config := &pipelineRunsConfigStruct{}

	// EXERCISE
	err := StartPipelineRun(ctx, mockPipelineRun, config)
	assert.NilError(t, err)

	// VERIFY
	// UpdateState should never be called
	mockPipelineRun.EXPECT().UpdateState(gomock.Any()).Times(0)
}

func Test_RunManager_StartPipelineRun_DoesCopySecret(t *testing.T) {
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
	ctx := mockContext(mockCtrl)
	mockPipelineRun := prepareMocksWithSpec(mockCtrl, spec)
	// UpdateState should never be called
	mockPipelineRun.EXPECT().
		UpdateState(gomock.Any()).
		Do(func(interface{}) { panic("unexpected call") }).
		AnyTimes()

	preparePredefinedClusterRole(t, ctx)
	config := &pipelineRunsConfigStruct{}

	// inject secret helper mock
	mockSecretHelper := secretMocks.NewMockSecretHelper(mockCtrl)
	testing := newRunManagerTestingWithRequiredStubs()
	testing.getSecretHelperStub = func(string, corev1.SecretInterface) secrets.SecretHelper {
		return mockSecretHelper
	}
	ctx = WithRunInstanceTesting(ctx, testing)

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
	err := StartPipelineRun(ctx, mockPipelineRun, config)
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
	mockPipelineRun := prepareMocksWithSpec(mockCtrl, spec)
	ctx := mockContext(mockCtrl)
	preparePredefinedClusterRole(t, ctx)
	config := &pipelineRunsConfigStruct{}
	ctx = WithRunInstanceTesting(ctx, newRunManagerTestingWithRequiredStubs())

	// EXPECT
	mockSecretProvider := secretMocks.NewMockSecretProvider(mockCtrl)
	mockSecretProvider.EXPECT().GetSecret(secretName).Return(nil, nil)
	ctx = secrets.WithSecretProvider(ctx, mockSecretProvider)

	mockPipelineRun.EXPECT().UpdateMessage(secrets.NewNotFoundError(secretName).Error())
	mockPipelineRun.EXPECT().UpdateResult(stewardv1alpha1.ResultErrorContent)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	err := StartPipelineRun(ctx, mockPipelineRun, config)
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
	mockPipelineRun := prepareMocksWithSpec(mockCtrl, spec)
	config := &pipelineRunsConfigStruct{}
	ctx := WithRunInstanceTesting(mockContext(mockCtrl), newRunManagerTestingWithRequiredStubs())
	preparePredefinedClusterRole(t, ctx)

	// EXPECT
	mockSecretProvider := secretMocks.NewMockSecretProvider(mockCtrl)
	mockSecretProvider.EXPECT().GetSecret(secretName).Return(nil, nil)
	ctx = secrets.WithSecretProvider(ctx, mockSecretProvider)

	mockPipelineRun.EXPECT().UpdateMessage(secrets.NewNotFoundError(secretName).Error())
	mockPipelineRun.EXPECT().UpdateResult(stewardv1alpha1.ResultErrorContent)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	err := StartPipelineRun(ctx, mockPipelineRun, config)
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
	mockPipelineRun := prepareMocksWithSpec(mockCtrl, spec)

	config := &pipelineRunsConfigStruct{}
	ctx := WithRunInstanceTesting(mockContext(mockCtrl), newRunManagerTestingWithRequiredStubs())
	preparePredefinedClusterRole(t, ctx)

	// EXPECT
	mockSecretProvider := secretMocks.NewMockSecretProvider(mockCtrl)
	mockSecretProvider.EXPECT().GetSecret(secretName).Return(nil, nil)
	ctx = secrets.WithSecretProvider(ctx, mockSecretProvider)
	mockPipelineRun.EXPECT().UpdateMessage(secrets.NewNotFoundError(secretName).Error())
	mockPipelineRun.EXPECT().UpdateResult(stewardv1alpha1.ResultErrorContent)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	err := StartPipelineRun(ctx, mockPipelineRun, config)
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
	mockPipelineRun := prepareMocksWithSpec(mockCtrl, spec)

	config := &pipelineRunsConfigStruct{}
	ctx := mockContext(mockCtrl)
	ctx = WithRunInstanceTesting(ctx, newRunManagerTestingWithRequiredStubs())
	preparePredefinedClusterRole(t, ctx)

	// EXPECT
	mockSecretProvider := secretMocks.NewMockSecretProvider(mockCtrl)
	mockSecretProvider.EXPECT().GetSecret(secretName).Return(nil, fmt.Errorf("Forbidden"))
	ctx = secrets.WithSecretProvider(ctx, mockSecretProvider)
	mockPipelineRun.EXPECT().UpdateMessage("Forbidden")
	mockPipelineRun.EXPECT().UpdateResult(stewardv1alpha1.ResultErrorInfra)
	mockPipelineRun.EXPECT().String() //logging

	// EXERCISE
	err := StartPipelineRun(ctx, mockPipelineRun, config)
	assert.Assert(t, err != nil)
}

func Test_RunManager_Cleanup_RemovesNamespace(t *testing.T) {
	t.Parallel()

	// SETUP
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	examinee := prepareMocks(mockCtrl)

	ctx := mockContext(mockCtrl)
	err := examinee.prepareRunNamespace(ctx)
	assert.NilError(t, err)

	preparePredefinedClusterRole(t, ctx)

	//TODO: mockNamespaceManager.EXPECT().Create()...

	// EXERCISE
	CleanupPipelineRun(ctx, examinee.pipelineRun)
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
		ctx context.Context, t *testing.T, pipelineRunJSON string,
	) (
		ctxOut context.Context, runInstanceObj *runInstance, cf *k8sfake.ClientFactory,
	) {
		pipelineRun := StewardObjectFromJSON(t, pipelineRunJSON).(*stewardv1alpha1.PipelineRun)
		t.Log("decoded:\n", spew.Sdump(pipelineRun))

		cf = k8sfake.NewClientFactory(
			k8sfake.Namespace("namespace1"),
			pipelineRun,
		)
		ctx = k8s.WithClientFactory(ctx, cf)
		k8sPipelineRun, err := k8s.NewPipelineRun(pipelineRun, cf)
		assert.NilError(t, err)
		ctx = secrets.WithSecretProvider(ctx, k8s.NewTenantNamespace(cf, pipelineRun.GetNamespace()).GetSecretProvider())
		ctx = WithRunInstanceTesting(ctx, newRunManagerTestingWithRequiredStubs())
		ctxOut = k8s.WithNamespaceManager(ctx, k8s.NewNamespaceManager(cf, "prefix1", 0))
		runInstanceObj = &runInstance{
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
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			ctx := mockServiceAccountTokenSecretRetriever(context.TODO(), mockCtrl)
			ctx, runInstance, cf := setupExaminee(ctx, t, pipelineRunJSON)
			ctx = WithRunInstanceTesting(ctx, newRunManagerTestingWithRequiredStubs())

			// exercise
			err = runInstance.createTektonTaskRun(ctx)
			assert.NilError(t, err)

			// verify
			taskRun := expectSingleTaskRun(t, cf, runInstance.pipelineRun)

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
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			ctx := mockServiceAccountTokenSecretRetriever(context.TODO(), mockCtrl)
			ctx, runInstance, cf := setupExaminee(ctx, t, pipelineRunJSON)

			// exercise
			err = runInstance.createTektonTaskRun(ctx)
			assert.NilError(t, err)

			// verify
			taskRun := expectSingleTaskRun(t, cf, runInstance.pipelineRun)

			param := findTaskRunParam(taskRun, TaskRunParamNameIndexURL)
			assert.Assert(t, param != nil)
			assert.Equal(t, tekton.ParamTypeString, param.Value.Type)
			assert.Equal(t, "", param.Value.StringVal)

			param = findTaskRunParam(taskRun, TaskRunParamNameRunIDJSON)
			assert.Assert(t, is.Nil(param))
		})
	}
}

func preparePredefinedClusterRole(t *testing.T, ctx context.Context) {
	// Create expected cluster role
	factory := k8s.GetClientFactory(ctx)
	_, err := factory.RbacV1beta1().ClusterRoles().Create(k8sfake.ClusterRole(string(runClusterRoleName)))
	assert.NilError(t, err)
}

func prepareMocks(ctrl *gomock.Controller) *runInstance {
	config := pipelineRunsConfigStruct{}
	return &runInstance{pipelineRun: prepareMocksWithSpec(ctrl, nil),
		pipelineRunsConfig: config,
	}
}

func mockContext(ctrl *gomock.Controller) context.Context {
	ctx := context.TODO()
	return mockFactories(ctx, ctrl)
}

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

func mockServiceAccountTokenSecretRetriever(ctx context.Context, ctrl *gomock.Controller) context.Context {
	mock := mocks.NewMockServiceAccountTokenSecretRetriever(ctrl)
	mock.EXPECT().ForObj(gomock.Any(), gomock.Any()).Return(&corev1api.Secret{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}, nil).AnyTimes()
	return k8s.WithServiceAccountTokenSecretRetriever(ctx, mock)
}

func prepareMocksWithSpec(ctrl *gomock.Controller, spec *stewardv1alpha1.PipelineSpec) *mocks.MockPipelineRun {
	if spec == nil {
		spec = &stewardv1alpha1.PipelineSpec{}
	}
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

	return mockPipelineRun
}
