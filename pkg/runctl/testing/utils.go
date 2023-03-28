package testing

import (
	"fmt"
	"testing"
	"time"

	stewardScheme "github.com/SAP/stewardci-core/pkg/client/clientset/versioned/scheme"
	k8sfake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	"github.com/SAP/stewardci-core/pkg/runctl/constants"
	tektonScheme "github.com/SAP/stewardci-core/pkg/tektonclient/clientset/versioned/scheme"
	"gotest.tools/v3/assert/cmp"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sScheme "k8s.io/client-go/kubernetes/scheme"
)

// StewardObjectFromJSON decodes a Steward object from its JSON representation.
// Panics in case of errors.
func StewardObjectFromJSON(t *testing.T, doc string) runtime.Object {
	versions := []schema.GroupVersion{
		{Group: "steward.sap.com", Version: "v1alpha1"},
	}
	decoder := stewardScheme.Codecs.UniversalDecoder(versions...)
	obj, _, err := decoder.Decode([]byte(doc), nil, nil)
	if err != nil {
		panic(err)
	}
	return obj
}

// TektonObjectFromJSON decodes a Tekton Pipeline object from its JSON
// representation.
// Panics in case of errors.
func TektonObjectFromJSON(t *testing.T, doc string) runtime.Object {
	versions := []schema.GroupVersion{
		{Group: "tekton.dev", Version: "v1beta1"},
	}
	decoder := tektonScheme.Codecs.UniversalDecoder(versions...)
	obj, _, err := decoder.Decode([]byte(doc), nil, nil)
	if err != nil {
		panic(err)
	}
	return obj
}

// CoreV1ObjectFromJSON decodes a Kubernetes Core v1 object from its JSON
// representation.
// Panics in case of errors.
func CoreV1ObjectFromJSON(t *testing.T, doc string) runtime.Object {
	decoder := k8sScheme.Codecs.UniversalDecoder(schema.GroupVersion{Version: "v1"})
	obj, _, err := decoder.Decode([]byte(doc), nil, nil)
	if err != nil {
		panic(err)
	}
	return obj
}

// TimeEqual compares an actual "k8s.io/apimachinery/pkg/apis/meta/v1".Time
// to an RFC3339-formatted timestamp string. It succeeds of both timestamps
// denote the same instant.
func TimeEqual(expectedAsRFC3339 string, actual metav1.Time) cmp.Comparison {
	return func() cmp.Result {
		expected, err := time.Parse(time.RFC3339, expectedAsRFC3339)
		if err != nil {
			panic(err)
		}
		expected = expected.UTC()
		actualU := actual.Time.UTC()
		if !expected.Equal(actualU) {
			return cmp.ResultFailure(fmt.Sprintf(
				"unexpected timestamp:\n"+
					"  expected: %s\n"+
					"  actual  : %s",
				expected.Format(time.RFC3339),
				actualU.Format(time.RFC3339),
			))
		}
		return cmp.ResultSuccess
	}
}

// FakeClusterRole creates a fake role for testing
func FakeClusterRole() *rbacv1.ClusterRole {
	return k8sfake.ClusterRole(string(constants.RunClusterRoleName))
}
