package stewardlabels

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert/cmp"

	"github.com/mohae/deepcopy"
	"gotest.tools/v3/assert"
)

type PropagateSuccessTestCase struct {
	// fields must be exported to make deepcopy work

	PreexistingDestLabels map[string]string
	SourceLabels          map[string]string
	LabelSpec             map[string]string

	ExpectedResultLabels map[string]string
}

type PropagateSuccessTestCases []PropagateSuccessTestCase

func (testcases PropagateSuccessTestCases) run(t *testing.T, namePrefix string) {
	t.Helper()

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%s%d", namePrefix, idx), func(t *testing.T) {
			t.Helper()

			// deepcopy removes shared parts between test cases and between fields of single test cases
			tc := deepcopy.Copy(tc).(PropagateSuccessTestCase)
			t.Parallel()
			logTestCaseSpec(t, tc)

			// SETUP
			destObj := &DummyObject1{}
			destObj.SetLabels(tc.PreexistingDestLabels)

			sourceObj := &DummyObject1{}
			sourceObj.SetLabels(tc.SourceLabels)

			// EXERCISE
			resultErr := propagate(destObj, sourceObj, tc.LabelSpec)

			// VERIFY
			assert.NilError(t, resultErr)
			assert.DeepEqual(t, tc.ExpectedResultLabels, destObj.GetLabels())

			if tc.PreexistingDestLabels != nil {
				assert.Assert(t, testableLabelMap(destObj.GetLabels()).IsSameMapAs(tc.PreexistingDestLabels),
					"label map of destination object has been replaced")
			}
		})
	}
}

func Test__propagate__noPropagationOrEnforcedValues(t *testing.T) {
	const (
		some1    = "some1key"
		some1Val = "some1Val"

		some2    = "some2key"
		some2Val = "some2Val"

		empty    = ""
		emptyVal = "emptyVal"
	)

	testcases := PropagateSuccessTestCases{}

	// cases without any propagation or enforced values
	for _, destLabels := range []map[string]string{
		nil,
		{},
		{
			some1: some1Val + "-dest",
		},
		{
			some1: some1Val + "-dest",
			some2: some2Val + "-dest",
		},
		{
			empty: emptyVal + "-dest",
		},
	} {
		for _, sourceLabels := range []map[string]string{
			nil,
			{},
			{
				some1: some1Val,
			},
			{
				some1: some1Val,
				some2: some2Val,
			},
			{
				empty: emptyVal,
			},
		} {
			for _, labelSpec := range []map[string]string{
				nil,
				{},
				{
					"nonexistentKey1": "",
				},
			} {
				testcases = append(testcases, PropagateSuccessTestCase{
					PreexistingDestLabels: destLabels,
					SourceLabels:          sourceLabels,
					LabelSpec:             labelSpec,
					ExpectedResultLabels:  destLabels, // unmodified
				})
			}
		}
	}

	// execute
	testcases.run(t, "")
}

func Test__propagate__Propagations(t *testing.T) {
	const (
		propa1    = "propa1Key"
		propa1Val = "propa1Val"

		propa2    = "propa2Key"
		propa2Val = "propa2Val"

		other1    = "other1Key"
		other1Val = "other1Val"

		empty    = ""
		emptyVal = "emptyVal"
	)

	// cases without additional labels on destObj
	{
		testcases := PropagateSuccessTestCases{}

		labelSpec := map[string]string{
			propa1: "",
			propa2: "",
			empty:  "",
		}
		expectedDestLabels := map[string]string{
			propa1: propa1Val,
			propa2: propa2Val,
			empty:  emptyVal,
		}

		for _, destLabels := range []map[string]string{
			nil,
			{},
			{
				propa1: propa1Val,
			},
			{
				propa1: propa1Val,
				propa2: propa2Val,
			},
			{
				empty: emptyVal,
			},
		} {
			for _, sourceLabels := range []map[string]string{
				{
					propa1: propa1Val,
					propa2: propa2Val,
					empty:  emptyVal,
				},
				{
					propa1: propa1Val,
					propa2: propa2Val,
					empty:  emptyVal,
					other1: other1Val,
				},
			} {
				testcases = append(testcases, PropagateSuccessTestCase{
					PreexistingDestLabels: destLabels,
					SourceLabels:          sourceLabels,
					LabelSpec:             labelSpec,
					ExpectedResultLabels:  expectedDestLabels,
				})
			}
		}

		// execute
		testcases.run(t, "WithoutAdditionalLabelInDestObj/")
	}

	// cases with additional nonpropagated label on destObj
	{
		testcases := PropagateSuccessTestCases{}

		labelSpec := map[string]string{
			propa1: "",
			propa2: "",
		}
		expectedDestLabels := map[string]string{
			propa1: propa1Val,
			propa2: propa2Val,
			other1: other1Val + "-dest",
		}

		for _, destLabels := range []map[string]string{
			{
				propa1: propa1Val,
				other1: other1Val + "-dest",
			},
			{
				propa1: propa1Val,
				propa2: propa2Val,
				other1: other1Val + "-dest",
			},
		} {
			for _, sourceLabels := range []map[string]string{
				{
					propa1: propa1Val,
					propa2: propa2Val,
				},
				{
					propa1: propa1Val,
					propa2: propa2Val,
					other1: other1Val,
				},
			} {
				testcases = append(testcases, PropagateSuccessTestCase{
					PreexistingDestLabels: destLabels,
					SourceLabels:          sourceLabels,
					LabelSpec:             labelSpec,
					ExpectedResultLabels:  expectedDestLabels,
				})
			}
		}

		// execute
		testcases.run(t, "WithAdditionalLabelInDestObj/")
	}
}

func Test__propagate__EnforcedValues(t *testing.T) {
	const (
		enfor1    = "enfor1Key"
		enfor1Val = "enfor1Val"

		enfor2    = "enfor2Key"
		enfor2Val = "enfor2Val"

		other1    = "other1Key"
		other1Val = "other1Val"

		empty    = ""
		emptyVal = "emptyVal"
	)

	// cases without additional labels on destObj
	{
		testcases := PropagateSuccessTestCases{}

		labelSpec := map[string]string{
			enfor1: enfor1Val,
			enfor2: enfor2Val,
			empty:  emptyVal,
		}
		expectedDestLabels := map[string]string{
			enfor1: enfor1Val,
			enfor2: enfor2Val,
			empty:  emptyVal,
		}
		for _, destLabels := range []map[string]string{
			nil,
			{},
			{
				enfor1: enfor1Val,
			},
			{
				enfor1: enfor1Val,
				enfor2: enfor2Val,
			},
			{
				empty: emptyVal,
			},
		} {
			for _, sourceLabels := range []map[string]string{
				nil,
				{},
				{
					other1: other1Val,
				},
				{
					enfor1: enfor1Val,
					other1: other1Val,
				},
				{
					enfor1: enfor1Val,
					enfor2: enfor2Val,
					other1: other1Val,
				},
				{
					empty: emptyVal,
				},
			} {
				testcases = append(testcases, PropagateSuccessTestCase{
					PreexistingDestLabels: destLabels,
					SourceLabels:          sourceLabels,
					LabelSpec:             labelSpec,
					ExpectedResultLabels:  expectedDestLabels,
				})
			}
		}

		// execute
		testcases.run(t, "WithoutAdditionalLabelInDestObj/")
	}

	// cases with additional nonpropagated label on destObj
	{
		testcases := PropagateSuccessTestCases{}

		labelSpec := map[string]string{
			enfor1: enfor1Val,
			enfor2: enfor2Val,
		}
		expectedDestLabels := map[string]string{
			enfor1: enfor1Val,
			enfor2: enfor2Val,
			other1: other1Val + "-dest",
		}

		for _, destLabels := range []map[string]string{
			{
				enfor1: enfor1Val,
				other1: other1Val + "-dest",
			},
			{
				enfor1: enfor1Val,
				enfor2: enfor2Val,
				other1: other1Val + "-dest",
			},
		} {
			for _, sourceLabels := range []map[string]string{
				nil,
				{},
				{
					other1: other1Val,
				},
				{
					enfor1: enfor1Val,
					other1: other1Val,
				},
				{
					enfor1: enfor1Val,
					enfor2: enfor2Val,
					other1: other1Val,
				},
			} {
				testcases = append(testcases, PropagateSuccessTestCase{
					PreexistingDestLabels: destLabels,
					SourceLabels:          sourceLabels,
					LabelSpec:             labelSpec,
					ExpectedResultLabels:  expectedDestLabels,
				})
			}
		}

		// execute
		testcases.run(t, "WithAdditionalLabelInDestObj/")
	}
}

type PropagateFailsTestCase struct {
	// fields must be exported to make deepcopy work

	DestLabels   map[string]string
	SourceLabels map[string]string
	LabelSpec    map[string]string

	ExpectedErrorMsg string
}

type PropagateFailsTestCases []PropagateFailsTestCase

func (testcases PropagateFailsTestCases) run(t *testing.T) {
	t.Helper()

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			t.Helper()

			tc := deepcopy.Copy(tc).(PropagateFailsTestCase)
			t.Parallel()
			logTestCaseSpec(t, tc)

			// SETUP
			destLabels := deepcopy.Copy(tc.DestLabels).(map[string]string)
			destObj := &DummyObject1{}
			destObj.SetLabels(destLabels)

			sourceObj := &DummyObject1{}
			sourceObj.SetLabels(tc.SourceLabels)

			// EXERCISE
			resultErr := propagate(destObj, sourceObj, tc.LabelSpec)

			// VERIFY
			assert.Error(t, resultErr, tc.ExpectedErrorMsg)
			assert.Assert(t, cmp.DeepEqual(tc.DestLabels, destObj.GetLabels()),
				"labels have been modified")
			if destLabels != nil {
				assert.Assert(t, testableLabelMap(destObj.GetLabels()).IsSameMapAs(destLabels),
					"label map of destination object has been replaced")
			}
		})
	}
}

func expectedErrorMessageForValueConflictAtDestObj(labelKey, existingValue, expectedValue string) string {
	return fmt.Sprintf(
		"value conflict: destination object label %q has existing value %q but %q is expected",
		labelKey, existingValue, expectedValue,
	)
}

func expectedErrorMessageForValueConflictAtSourceObj(labelKey, existingValue, expectedValue string) string {
	return fmt.Sprintf(
		"value conflict: source object label %q has value %q but %q is expected",
		labelKey, existingValue, expectedValue,
	)
}

func Test__propagate__Fails_ValueConflictOnDestObj_FromPropagation(t *testing.T) {
	const (
		conflict1    = "conflict1Key"
		conflict1Val = "conflict1Val"
	)

	testcases := PropagateFailsTestCases{}

	testcases = append(testcases, PropagateFailsTestCase{
		DestLabels: map[string]string{
			conflict1: conflict1Val + "-dest",
		},
		SourceLabels: map[string]string{
			conflict1: conflict1Val + "-source",
		},
		LabelSpec: map[string]string{
			conflict1: "",
		},

		ExpectedErrorMsg: expectedErrorMessageForValueConflictAtDestObj(
			conflict1, conflict1Val+"-dest", conflict1Val+"-source",
		),
	})

	testcases.run(t)
}

func Test__propagate__Fails_ValueConflictOnDestObj_FromEnforcedValue(t *testing.T) {
	const (
		conflict1    = "conflict1Key"
		conflict1Val = "conflict1Val"
	)

	testcases := PropagateFailsTestCases{}

	testcases = append(testcases, PropagateFailsTestCase{
		DestLabels: map[string]string{
			conflict1: conflict1Val,
		},
		SourceLabels: nil,
		LabelSpec: map[string]string{
			conflict1: conflict1Val + "-enforced",
		},

		ExpectedErrorMsg: expectedErrorMessageForValueConflictAtDestObj(
			conflict1, conflict1Val, conflict1Val+"-enforced",
		),
	})

	testcases.run(t)
}

func Test__propagate__Fails_ValueConflictOnSourceObj(t *testing.T) {
	const (
		conflict1    = "conflict1Key"
		conflict1Val = "conflict1Val"
	)

	testcases := PropagateFailsTestCases{}

	testcases = append(testcases, PropagateFailsTestCase{
		DestLabels: nil,
		SourceLabels: map[string]string{
			conflict1: conflict1Val,
		},
		LabelSpec: map[string]string{
			conflict1: conflict1Val + "-enforced",
		},

		ExpectedErrorMsg: expectedErrorMessageForValueConflictAtSourceObj(
			conflict1, conflict1Val, conflict1Val+"-enforced",
		),
	})

	testcases.run(t)
}
