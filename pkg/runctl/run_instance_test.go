package runctl

import (
	"context"
	"strings"
	"testing"

	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s"
	fake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	k8sfake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	mocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"gotest.tools/assert"
	corev1api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
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
	ctx = WithRunInstanceTesting(ctx, newRunManagerTestingWithAllNoopStubs())

	return mockFactories(ctx, ctrl)
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
