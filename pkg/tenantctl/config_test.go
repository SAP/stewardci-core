package tenantctl

import (
	"math"
	"strconv"
	"testing"

	assert "gotest.tools/assert"
	cmp "gotest.tools/assert/cmp"
	is "gotest.tools/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

/*
 * Within this file the annotation keys are written as string literals instead
 * of using the respective constants from the Steward API package.
 * The reason is that tests should fail in case the constants are changed
 * (incompatible API change).
 */

func Test_tenantNamespaceSuffixLengthDefault_isLessThanOrEqualTo_tenantNamespaceSuffixLengthMax(t *testing.T) {
	assert.Assert(t, tenantNamespaceSuffixLengthDefault <= tenantNamespaceSuffixLengthMax)
}

func Test_clientConfigImpl_GetTenantNamespaceSuffixLength(t *testing.T) {
	for _, tc := range []struct {
		inputValue          int64
		expectedOutputValue uint8
	}{
		{math.MinInt64, tenantNamespaceSuffixLengthDefault},
		{-1, tenantNamespaceSuffixLengthDefault},
		{0, 0},
		{1, 1},
		{int64(tenantNamespaceSuffixLengthDefault), tenantNamespaceSuffixLengthDefault},
		{int64(tenantNamespaceSuffixLengthMax) - 1, tenantNamespaceSuffixLengthMax - 1},
		{int64(tenantNamespaceSuffixLengthMax), tenantNamespaceSuffixLengthMax},
		{int64(tenantNamespaceSuffixLengthMax) + 1, tenantNamespaceSuffixLengthMax},
		{math.MaxInt64, tenantNamespaceSuffixLengthMax},
	} {
		t.Run(strconv.FormatInt(tc.inputValue, 10), func(t *testing.T) {
			// SETUP
			config := clientConfigImpl{tenantNamespaceSuffixLength: tc.inputValue}

			// EXERCISE
			outputValue := config.GetTenantNamespaceSuffixLength()

			// VERIFY
			assert.Equal(t, tc.expectedOutputValue, outputValue)
		})
	}
}

func Test_getClientConfig_ReturnsValuesFromAnnotations(t *testing.T) {
	// SETUP
	const configuredRandomLength int64 = 10

	cf := createLister(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "Client1",
				Annotations: map[string]string{
					"steward.sap.com/tenant-namespace-prefix":        "testprefix",
					"steward.sap.com/tenant-namespace-suffix-length": strconv.FormatInt(configuredRandomLength, 10),
					"steward.sap.com/tenant-role":                    "testrole",
				},
			},
		},
	)

	// EXERCISE
	config, err := getClientConfig(cf, "Client1")

	// VERIFY
	assert.NilError(t, err)
	assert.Equal(t, configuredRandomLength, config.(*clientConfigImpl).tenantNamespaceSuffixLength)
	prefix := config.GetTenantNamespacePrefix()
	assert.Equal(t, "testprefix", prefix)
	tenantRole := config.GetTenantRoleName()
	assert.Equal(t, "testrole", string(tenantRole))
}

func Test_getClientConfig_NamespaceParameterIsZeroLengthString(t *testing.T) {
	// SETUP
	const emptyNameString = ""

	cf := createLister(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "Client1",
				Annotations: map[string]string{
					"steward.sap.com/tenant-namespace-prefix":        "testprefix",
					"steward.sap.com/tenant-namespace-suffix-length": "10",
					"steward.sap.com/tenant-role":                    "testrole",
				},
			},
		},
	)

	// EXERCISE
	assert.Assert(t, cmp.Panics(func() {
		getClientConfig(cf, emptyNameString)
	}))
}

func Test_getClientConfig_ClientNamespaceNotExisting(t *testing.T) {
	// SETUP
	cf := createLister(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "Namespace1",
				Annotations: map[string]string{},
			},
		},
	)

	// EXERCISE
	_, err := getClientConfig(cf, "NotExistingClientNamespace")

	// VERIFY
	assert.Assert(t, err != nil)
	assert.Assert(t, is.Regexp("^could not get namespace 'NotExistingClientNamespace': .*", err.Error()))
}

func Test_getClientConfig_AnnotationTenantNamespacePrefix_Missing(t *testing.T) {
	// SETUP
	cf := createLister(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "Client1",
				Annotations: map[string]string{
					//"steward.sap.com/tenant-namespace-prefix":    "testprefix",
					"steward.sap.com/tenant-namespace-suffix-length": "10",
					"steward.sap.com/tenant-role":                    "testrole",
				},
			},
		},
	)

	// EXERCISE
	_, err := getClientConfig(cf, "Client1")

	// VERIFY
	assert.Assert(t, err != nil)
	assert.Equal(t,
		"annotation 'steward.sap.com/tenant-namespace-prefix' is missing"+
			" on client namespace 'Client1'",
		err.Error(),
	)
}

func Test_getClientConfig_AnnotationTenantNamespacePrefix_EmptyValue(t *testing.T) {
	// SETUP
	cf := createLister(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "Client1",
				Annotations: map[string]string{
					"steward.sap.com/tenant-namespace-prefix":        "", // <== empty value!
					"steward.sap.com/tenant-namespace-suffix-length": "10",
					"steward.sap.com/tenant-role":                    "testrole",
				},
			},
		},
	)

	// EXERCISE
	_, err := getClientConfig(cf, "Client1")

	// VERIFY
	assert.Assert(t, err != nil)
	assert.Equal(t,
		"annotation 'steward.sap.com/tenant-namespace-prefix' on client namespace"+
			" 'Client1' must not have an empty value",
		err.Error(),
	)
}

func Test_getClientConfig_AnnotationTenantRole_Missing(t *testing.T) {
	// SETUP
	cf := createLister(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "Client1",
				Annotations: map[string]string{
					"steward.sap.com/tenant-namespace-prefix":        "testprefix",
					"steward.sap.com/tenant-namespace-suffix-length": "10",
					//"steward.sap.com/tenant-role":                  "testrole",
				},
			},
		},
	)

	// EXERCISE
	_, err := getClientConfig(cf, "Client1")

	// VERIFY
	assert.Assert(t, err != nil)
	assert.Equal(t, "annotation 'steward.sap.com/tenant-role' is missing on client namespace 'Client1'", err.Error())
}

func Test_getClientConfig_AnnotationTenantRole_EmptyValue(t *testing.T) {
	// SETUP
	cf := createLister(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "Client1",
				Annotations: map[string]string{
					"steward.sap.com/tenant-namespace-prefix":        "testprefix",
					"steward.sap.com/tenant-namespace-suffix-length": "10",
					"steward.sap.com/tenant-role":                    "", // <== empty value!
				},
			},
		},
	)

	// EXERCISE
	_, err := getClientConfig(cf, "Client1")

	// VERIFY
	assert.Assert(t, err != nil)
	assert.Equal(t,
		"annotation 'steward.sap.com/tenant-role' on client namespace 'Client1'"+
			" must not have an empty value",
		err.Error(),
	)
}

func Test_getClientConfig_AnnotationTenantNamespaceSuffixLength_Missing(t *testing.T) {
	// SETUP
	cf := createLister(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "Client1",
				Annotations: map[string]string{
					"steward.sap.com/tenant-namespace-prefix": "testprefix",
					//"steward.sap.com/tenant-namespace-suffix-length": "10",
					"steward.sap.com/tenant-role": "testrole",
				},
			},
		},
	)

	// EXERCISE
	config, _ := getClientConfig(cf, "Client1")

	// VERIFY
	value := config.(*clientConfigImpl).tenantNamespaceSuffixLength
	assert.Equal(t, int64(-1), value)
}

func Test_getClientConfig_AnnotationTenantNamespaceSuffixLength_InvalidValue(t *testing.T) {
	for num, value := range []string{
		// not an integer:
		"",
		"a",
		"-",
		"+",
		"7a",
		// out of int8 range:
		strconv.Itoa(math.MinInt8 - 1),
		strconv.Itoa(math.MaxInt8 + 1),
	} {
		t.Run(strconv.Itoa(num)+"_"+value, func(t *testing.T) {
			// SETUP
			cf := createLister(
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "Client1",
						Annotations: map[string]string{
							"steward.sap.com/tenant-namespace-prefix":        "testprefix",
							"steward.sap.com/tenant-namespace-suffix-length": value,
							"steward.sap.com/tenant-role":                    "testrole",
						},
					},
				},
			)

			// EXERCISE
			_, err := getClientConfig(cf, "Client1")

			// VERIFY
			assert.Assert(t, err != nil)
			assert.Equal(t,
				"annotation 'steward.sap.com/tenant-namespace-suffix-length' on client namespace"+
					" 'Client1' has an invalid value: '"+value+"':"+
					" should be a decimal integer in the range of [-128, 127]",
				err.Error(),
			)
		})
	}
}

func Test_getClientConfig_AnnotationTenantNamespaceSuffixLength_ValidValue(t *testing.T) {
	for num, tc := range []struct {
		value       string
		expectedLen int64
	}{
		{"0", 0},
		{"+0", 0},
		{"-0", 0},
		{"1", 1},
		{"+1", 1},
		{"-1", -1},
		{strconv.Itoa(math.MinInt8), math.MinInt8},
		{strconv.Itoa(math.MaxInt8), math.MaxInt8},
	} {
		t.Run(strconv.Itoa(num)+"_"+tc.value, func(t *testing.T) {
			// SETUP
			cf := createLister(
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "Client1",
						Annotations: map[string]string{
							"steward.sap.com/tenant-namespace-prefix":        "testprefix",
							"steward.sap.com/tenant-namespace-suffix-length": tc.value,
							"steward.sap.com/tenant-role":                    "testrole",
						},
					},
				},
			)

			// EXERCISE
			config, err := getClientConfig(cf, "Client1")

			// VERIFY
			assert.NilError(t, err)
			assert.Equal(t, tc.expectedLen, config.(*clientConfigImpl).tenantNamespaceSuffixLength)
		})
	}
}

func Test_getClientConfig_TwoClients(t *testing.T) {
	// SETUP
	cf := createLister(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "Client1",
				Annotations: map[string]string{
					"steward.sap.com/tenant-namespace-prefix":        "c1",
					"steward.sap.com/tenant-namespace-suffix-length": "6",
					"steward.sap.com/tenant-role":                    "r1",
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "Client2",
				Annotations: map[string]string{
					"steward.sap.com/tenant-namespace-prefix":        "c2",
					"steward.sap.com/tenant-namespace-suffix-length": "4",
					"steward.sap.com/tenant-role":                    "r2",
				},
			},
		},
	)

	// EXERCISE
	config1, err1 := getClientConfig(cf, "Client1")
	config2, err2 := getClientConfig(cf, "Client2")

	// VERIFY
	prefix1 := config1.GetTenantNamespacePrefix()
	prefix2 := config2.GetTenantNamespacePrefix()
	role1 := config1.GetTenantRoleName()
	role2 := config2.GetTenantRoleName()
	rand1 := config1.GetTenantNamespaceSuffixLength()
	rand2 := config2.GetTenantNamespaceSuffixLength()

	assert.NilError(t, err1)
	assert.NilError(t, err2)
	assert.Equal(t, "c1", prefix1)
	assert.Equal(t, "c2", prefix2)
	assert.Equal(t, "r1", string(role1))
	assert.Equal(t, "r2", string(role2))
	assert.Equal(t, uint8(6), rand1)
	assert.Equal(t, uint8(4), rand2)
}

func createLister(namespaces ...*corev1.Namespace) listers.NamespaceLister {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	for _, namespace := range namespaces {
		indexer.Add(namespace)
	}
	return listers.NewNamespaceLister(indexer)
}
