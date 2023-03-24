package runmgr

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	stewardfakeclient "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/fake"
	k8sfake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	k8smocks "github.com/SAP/stewardci-core/pkg/k8s/mocks"
	secretmocks "github.com/SAP/stewardci-core/pkg/k8s/secrets/mocks"
	cfg "github.com/SAP/stewardci-core/pkg/runctl/cfg"
	"github.com/SAP/stewardci-core/pkg/runctl/constants"
	runctltesting "github.com/SAP/stewardci-core/pkg/runctl/testing"
	tektonfakeclient "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned/fake"
	gomock "github.com/golang/mock/gomock"
	"github.com/lithammer/dedent"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	assert "gotest.tools/v3/assert"
	assertcmp "gotest.tools/v3/assert/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

type testHelper1 struct {
	t                   *testing.T
	ctx                 context.Context
	namespace1          string
	pipelineRun1        string
	runNamespace1       string
	tektonClientset     *tektonfakeclient.Clientset
	tektonTaskName      string
	tektonTaskNamespace string
}

func newTestHelper1(t *testing.T) *testHelper1 {
	h := &testHelper1{
		t:                   t,
		ctx:                 context.Background(),
		namespace1:          "namespace1",
		pipelineRun1:        "pipelinerun1",
		runNamespace1:       "runNamespace1",
		tektonClientset:     tektonfakeclient.NewSimpleClientset(),
		tektonTaskName:      "taskName1",
		tektonTaskNamespace: "taskNamespace1",
	}
	return h
}

func (h *testHelper1) runsConfigWithTaskData() *cfg.PipelineRunsConfigStruct {
	return &cfg.PipelineRunsConfigStruct{
		TektonTaskName:      h.tektonTaskName,
		TektonTaskNamespace: h.tektonTaskNamespace,
	}
}

func (h *testHelper1) getPipelineRunFromStorage(cf *k8sfake.ClientFactory, namespace, name string) *stewardv1alpha1.PipelineRun {
	t := h.t
	t.Helper()

	pipelineRun, err := cf.StewardV1alpha1().PipelineRuns(namespace).Get(h.ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("could not get pipeline run %q from namespace %q: %s", name, namespace, err.Error())
	}
	if pipelineRun == nil {
		t.Fatalf("could not get pipeline run %q from namespace %q: get was successfull but returned nil", name, namespace)
	}
	return pipelineRun
}

func (h *testHelper1) VerifyNamespace(cf *k8sfake.ClientFactory, nsName, expectedPurpose string, runNamespaceRandomLength int) {
	t := h.t
	t.Helper()

	namePattern := regexp.MustCompile(
		`^(steward-run-([[:alnum:]]{` + strconv.Itoa(runNamespaceRandomLength) + `})-` +
			regexp.QuoteMeta(expectedPurpose) +
			`-)[[:alnum:]]*$`)
	assert.Assert(t, assertcmp.Regexp(namePattern, nsName))

	namespace, err := cf.CoreV1().Namespaces().Get(h.ctx, nsName, metav1.GetOptions{})
	assert.NilError(t, err)
	assert.Equal(t, namespace.ObjectMeta.GenerateName, namePattern.FindStringSubmatch(nsName)[1])

	// labels
	{
		_, exists := namespace.GetLabels()[stewardv1alpha1.LabelSystemManaged]
		assert.Assert(t, exists)
	}
}

func (h *testHelper1) assertThatExactlyTheseNamespacesExist(cf *k8sfake.ClientFactory, expected ...string) {
	t := h.t
	t.Helper()

	list, err := cf.CoreV1().Namespaces().List(h.ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(err.Error())
	}
	actual := []string{}
	for _, item := range list.Items {
		if item.GetName() != "" {
			actual = append(actual, item.GetName())
		}
	}
	sort.Strings(actual)

	if expected == nil {
		expected = []string{}
	}
	{
		temp := []string{}
		for _, item := range expected {
			if item != "" {
				temp = append(temp, item)
			}
		}
		expected = temp
	}
	sort.Strings(expected)

	assert.DeepEqual(t, expected, actual)
}

func (h *testHelper1) preparePredefinedClusterRole(cf *k8smocks.MockClientFactory) {
	t := h.t
	t.Helper()

	_, err := cf.RbacV1().ClusterRoles().Create(h.ctx, runctltesting.FakeClusterRole(), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("could not create cluster role: %s", err.Error())
	}
}

func (h *testHelper1) addTektonTaskRun(cf *k8smocks.MockClientFactory) {
	t := h.t
	t.Helper()
	_, err := cf.TektonV1beta1().TaskRuns(h.runNamespace1).Create(h.ctx, h.dummyTektonTaskRun(), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("could not create tekton task run: %s", err.Error())
	}
}

func (h *testHelper1) dummyTektonTaskRun() *tektonv1beta1.TaskRun {
	t := h.t
	t.Helper()
	return &tektonv1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.TektonTaskRunName,
			Namespace: h.runNamespace1,
		},
	}
}

func (h *testHelper1) prepareMocks(ctrl *gomock.Controller) (*k8smocks.MockClientFactory, *k8smocks.MockPipelineRun, *secretmocks.MockSecretProvider) {
	return h.prepareMocksWithSpec(ctrl, &stewardv1alpha1.PipelineSpec{})
}

func (h *testHelper1) prepareMocksWithSpec(ctrl *gomock.Controller, spec *stewardv1alpha1.PipelineSpec) (*k8smocks.MockClientFactory, *k8smocks.MockPipelineRun, *secretmocks.MockSecretProvider) {
	mockFactory := k8smocks.NewMockClientFactory(ctrl)

	kubeClientSet := kubefake.NewSimpleClientset()
	kubeClientSet.PrependReactor("create", "*", k8sfake.GenerateNameReactor(0))

	mockFactory.EXPECT().CoreV1().Return(kubeClientSet.CoreV1()).AnyTimes()
	mockFactory.EXPECT().RbacV1().Return(kubeClientSet.RbacV1()).AnyTimes()
	mockFactory.EXPECT().NetworkingV1().Return(kubeClientSet.NetworkingV1()).AnyTimes()

	dynamicClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	mockFactory.EXPECT().Dynamic().Return(dynamicClient).AnyTimes()

	stewardClientset := stewardfakeclient.NewSimpleClientset()
	mockFactory.EXPECT().StewardV1alpha1().Return(stewardClientset.StewardV1alpha1()).AnyTimes()

	mockFactory.EXPECT().TektonV1beta1().Return(h.tektonClientset.TektonV1beta1()).AnyTimes()

	runNamespace := h.runNamespace1
	auxNamespace := ""
	mockPipelineRun := k8smocks.NewMockPipelineRun(ctrl)
	mockPipelineRun.EXPECT().GetAPIObject().Return(&stewardv1alpha1.PipelineRun{Spec: *spec}).AnyTimes()
	mockPipelineRun.EXPECT().GetSpec().Return(spec).AnyTimes()
	mockPipelineRun.EXPECT().GetStatus().Return(&stewardv1alpha1.PipelineStatus{}).AnyTimes()
	mockPipelineRun.EXPECT().GetKey().Return("key").AnyTimes()
	mockPipelineRun.EXPECT().GetValidatedJenkinsfileRepoServerURL().Return("server", nil).AnyTimes()
	mockPipelineRun.EXPECT().GetRunNamespace().DoAndReturn(func() string {
		return runNamespace
	}).AnyTimes()
	mockPipelineRun.EXPECT().GetAuxNamespace().DoAndReturn(func() string {
		return auxNamespace
	}).AnyTimes()

	mockPipelineRun.EXPECT().UpdateRunNamespace(gomock.Any()).Do(func(arg string) {
		runNamespace = arg
	}).MaxTimes(1)
	mockPipelineRun.EXPECT().UpdateAuxNamespace(gomock.Any()).Do(func(arg string) {
		auxNamespace = arg
	}).MaxTimes(1)
	mockPipelineRun.EXPECT().CommitStatus(gomock.Any()).MaxTimes(1)

	mockSecretProvider := secretmocks.NewMockSecretProvider(ctrl)

	return mockFactory, mockPipelineRun, mockSecretProvider
}

// fixIndent removes common leading whitespace from all lines
// and replaces all tabs by spaces
func fixIndent(s string) (out string) {
	const TAB = "   "
	out = s
	out = dedent.Dedent(out)
	out = strings.ReplaceAll(out, "\t", TAB)
	return
}
