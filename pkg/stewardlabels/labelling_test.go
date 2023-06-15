package stewardlabels

import (
	"fmt"
	"testing"

	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/mohae/deepcopy"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LabelAsSystemManagedTestCase struct {
	// fields must be exported to make deepcopy work
	PreexistingLabels    map[string]string
	ExpectedResultLabels map[string]string
}

type LabelAsSystemManagedTestCases []LabelAsSystemManagedTestCase

func (testcases LabelAsSystemManagedTestCases) run(t *testing.T, namePrefix string) {
	t.Helper()

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%s%d", namePrefix, idx), func(t *testing.T) {
			t.Helper()

			tc := deepcopy.Copy(tc).(LabelAsSystemManagedTestCase)
			t.Parallel()
			logTestCaseSpec(t, tc)

			// SETUP
			obj := &DummyObject1{}
			obj.SetLabels(tc.PreexistingLabels)

			// EXERCISE
			LabelAsSystemManaged(obj)

			// VERIFY
			assert.DeepEqual(t, tc.ExpectedResultLabels, obj.GetLabels())

			if tc.PreexistingLabels != nil {
				assert.Assert(t, testableLabelMap(obj.GetLabels()).IsSameMapAs(tc.PreexistingLabels),
					"label map of object has been replaced")
			}
		})
	}
}

func Test__LabelAsSystemManaged(t *testing.T) {
	const (
		some1    = "some1key"
		some1Val = "some1Val"
	)

	testcases := LabelAsSystemManagedTestCases{}

	// cases without additional labels
	{
		expectedResultLabels := map[string]string{
			stewardv1alpha1.LabelSystemManaged: "",
		}

		for _, preexistingLabels := range []map[string]string{
			nil,
			{},
			{stewardv1alpha1.LabelSystemManaged: ""},
			{stewardv1alpha1.LabelSystemManaged: "nonEmptyValue"},
		} {
			testcases = append(testcases, LabelAsSystemManagedTestCase{
				PreexistingLabels:    preexistingLabels,
				ExpectedResultLabels: expectedResultLabels,
			})
		}

		// execute
		testcases.run(t, "withoutAdditionalLabel/")
	}

	// cases with additional label
	{
		expectedResultLabels := map[string]string{
			some1:                              some1Val,
			stewardv1alpha1.LabelSystemManaged: "",
		}

		for _, existingLabels := range []map[string]string{
			{
				some1: some1Val,
			},
			{
				some1:                              some1Val,
				stewardv1alpha1.LabelSystemManaged: "",
			},
			{
				some1:                              some1Val,
				stewardv1alpha1.LabelSystemManaged: "nonEmptyValue",
			},
		} {
			testcases = append(testcases, LabelAsSystemManagedTestCase{
				PreexistingLabels:    existingLabels,
				ExpectedResultLabels: expectedResultLabels,
			})
		}

		// execute
		testcases.run(t, "withAdditionalLabel/")
	}
}

func Test__LabelAsSystemManaged__NilArg(t *testing.T) {
	// EXERCISE
	LabelAsSystemManaged(nil)
}

func Test__LabelAsOwnedByPipelineRun(t *testing.T) {
	const (
		ownerName      = "pipelinerun-1"
		ownerNamespace = "owner-1-namespace"
	)

	// SETUP
	obj := &DummyObject1{}

	owner := &stewardv1alpha1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ownerName,
			Namespace: ownerNamespace,
		},
	}

	// EXERCISE
	resultErr := LabelAsOwnedByPipelineRun(obj, owner)

	// VERIFY
	assert.NilError(t, resultErr)

	expectedResultLabels := map[string]string{
		stewardv1alpha1.LabelOwnerPipelineRunNamespace: ownerNamespace,
		stewardv1alpha1.LabelOwnerPipelineRunName:      ownerName,
	}
	assert.DeepEqual(t, expectedResultLabels, obj.GetLabels())
}

func Test__LabelAsOwnedByPipelineRun__NilArg__obj(t *testing.T) {
	// SETUP
	owner := &stewardv1alpha1.PipelineRun{}
	owner.SetName("name1")
	owner.SetNamespace("namespace1")

	// EXERCISE
	resultErr := LabelAsOwnedByPipelineRun(nil, owner)

	// VERIFY
	assert.NilError(t, resultErr)
}

func Test__LabelAsOwnedByPipelineRun__NilArg__owner(t *testing.T) {
	// SETUP
	obj := &DummyObject1{}
	obj.SetName("name1")

	// EXERCISE and VERIFY
	assert.Assert(t, cmp.Panics(func() {
		LabelAsOwnedByPipelineRun(obj, nil)
	}))
}
