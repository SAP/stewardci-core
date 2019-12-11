package builder

import (
	"testing"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Tenant(t *testing.T) {
	tenant := Tenant("bar")
	expectedtenant := &api.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    "bar",
			GenerateName: "t-",
		},
	}
	assert.DeepEqual(t, expectedtenant, tenant)
}

func Test_TenantFixName(t *testing.T) {
	tenant := TenantFixName("foo", "bar")
	expected := &api.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "bar",
			Name:      "foo",
		},
	}
	assert.DeepEqual(t, expected, tenant)
}
