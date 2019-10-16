package tenantctl

import (
	"fmt"
	"testing"

	fake "github.com/SAP/stewardci-core/pkg/k8s/fake"
	assert "gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Config(t *testing.T) {
	const configuredRandomLength = 10
	const anyOtherNumber = 7

	cf := fake.NewClientFactory(
		&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "Client1", Annotations: map[string]string{
			"tenant-namespace-prefix":    "testprefix",
			"tenant-role":                "testrole",
			"tenant-random-length-bytes": fmt.Sprintf("%v", configuredRandomLength),
		}}},
	)

	config, err := getConfig(cf, "Client1")

	assert.NilError(t, err)
	assert.Equal(t, configuredRandomLength, config.GetRandomLengthBytesOrDefault(anyOtherNumber))
	prefix := config.GetTenantNamespacePrefix()
	assert.Equal(t, "testprefix", prefix)
	tenantRole := config.GetTenantRoleName()
	assert.Equal(t, "testrole", fmt.Sprintf("%v", tenantRole))
}

func Test_EmptyNamespaceParameter(t *testing.T) {
	const emptyNameString = ""

	cf := fake.NewClientFactory(
		&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "Client1", Annotations: map[string]string{}}},
	)

	_, err := getConfig(cf, emptyNameString)

	assert.Assert(t, err != nil)
	assert.Equal(t, "GetConfig failed - client namespace not specified", err.Error())
}

func Test_MissingClientNamespace(t *testing.T) {
	cf := fake.NewClientFactory(
		&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "Namespace1", Annotations: map[string]string{}}},
	)

	_, err := getConfig(cf, "OtherNamespaceName")

	assert.Assert(t, err != nil)
	assert.Equal(t, "GetConfig failed - could not get namespace: namespaces \"OtherNamespaceName\" not found", err.Error())
}

func Test_MissingPrefix(t *testing.T) {
	cf := fake.NewClientFactory(
		&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "Client1", Annotations: map[string]string{
			//"tenant-namespace-prefix":    "testprefix",
			"tenant-role":                "testrole",
			"tenant-random-length-bytes": "10",
		}}},
	)

	_, err := getConfig(cf, "Client1")

	assert.Assert(t, err != nil)
	assert.Equal(t, "tenant-namespace-prefix not configured for client", err.Error())
}

func Test_MissingRole(t *testing.T) {
	cf := fake.NewClientFactory(
		&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "Client1", Annotations: map[string]string{
			"tenant-namespace-prefix": "testprefix",
			//"tenant-role":                "testrole",
			"tenant-random-length-bytes": "10",
		}}},
	)

	_, err := getConfig(cf, "Client1")

	assert.Assert(t, err != nil)
	assert.Equal(t, "tenant-role not configured for client", err.Error())
}

func Test_MissingRandom(t *testing.T) {
	cf := fake.NewClientFactory(
		&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "Client1", Annotations: map[string]string{
			"tenant-namespace-prefix": "testprefix",
			"tenant-role":             "testrole",
			//"tenant-random-length-bytes": "10",
		}}},
	)

	config, _ := getConfig(cf, "Client1")

	value := config.GetRandomLengthBytesOrDefault(9)
	assert.Equal(t, 9, value)
}

func Test_TwoClients(t *testing.T) {
	cf := fake.NewClientFactory(
		&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "Client1", Annotations: map[string]string{
			"tenant-namespace-prefix":    "c1",
			"tenant-role":                "r1",
			"tenant-random-length-bytes": "6",
		}}},
		&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "Client2", Annotations: map[string]string{
			"tenant-namespace-prefix":    "c2",
			"tenant-role":                "r2",
			"tenant-random-length-bytes": "4",
		}}},
	)

	config1, err1 := getConfig(cf, "Client1")
	config2, err2 := getConfig(cf, "Client2")

	prefix1 := config1.GetTenantNamespacePrefix()
	prefix2 := config2.GetTenantNamespacePrefix()
	role1 := config1.GetTenantRoleName()
	role2 := config2.GetTenantRoleName()
	rand1 := config1.GetRandomLengthBytesOrDefault(0)
	rand2 := config2.GetRandomLengthBytesOrDefault(0)

	assert.NilError(t, err1)
	assert.NilError(t, err2)
	assert.Equal(t, "c1", prefix1)
	assert.Equal(t, "c2", prefix2)
	assert.Equal(t, "r1", string(role1))
	assert.Equal(t, "r2", string(role2))
	assert.Equal(t, 6, rand1)
	assert.Equal(t, 4, rand2)
}
