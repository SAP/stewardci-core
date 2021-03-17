package stewardlabels

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"gotest.tools/assert/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DummyObject1 struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func logTestCaseSpec(t *testing.T, spec interface{}) {
	t.Helper()

	spewConf := &spew.ConfigState{
		Indent:                  "\t",
		DisableCapacities:       true,
		DisablePointerAddresses: true,
	}
	t.Logf("Testcase:\n%s", spewConf.Sdump(spec))
}

type testableLabelMap map[string]string

func (actual testableLabelMap) IsSameMapAs(expected map[string]string) cmp.Comparison {
	return func() cmp.Result {
		if result := cmp.DeepEqual(expected, map[string]string(actual))(); !result.Success() {
			return result
		}

		if expected == nil && actual != nil {
			return cmp.ResultFailure("expected nil but was not nil")
		}

		// Cannot test directly whether two maps reference the same map
		// object internally (A == B is not allowed).
		// Test indirectly by inserting into A and expecting to see it in B.
		if expected != nil {
			testKey := uuid.New().String()
			expected[testKey] = testKey
			found := actual[testKey] == testKey
			delete(expected, testKey)
			if !found {
				return cmp.ResultFailure("maps are not the same")
			}
		}
		return cmp.ResultSuccess
	}
}
