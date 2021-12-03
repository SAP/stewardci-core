package k8s

import (
	"context"
	"strconv"
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_NewNamespaceManager(t *testing.T) {
	// SETUP
	cf := fake.NewClientFactory()

	// EXERCISE
	result := NewNamespaceManager(cf, "prefix1", 255)

	// VERIFY
	assert.Assert(t, result != nil)
	impl := result.(*namespaceManager)
	assert.Assert(t, impl.nsInterface != nil)
	assert.Equal(t, "prefix1", impl.prefix)
	assert.Equal(t, uint8(255), impl.suffixLength)
}

func Test_namespaceManager_generateName(t *testing.T) {
	for ti, tc := range []struct {
		prefix               string
		customPart           string
		suffixLength         uint8
		expectedResultRegexp string
	}{
		{"", "", 0, ""},
		{"a", "", 0, "^a$"},
		{"-", "", 0, "^-$"},

		{"", "b", 0, "^b$"},
		{"", "-", 0, "^-$"},
		{"a", "b", 0, "^a-b$"},
		{"-", "-", 0, "^---$"},

		{"", "", 3, "^[0-9a-z]{3}$"},
		{"", "", 100, "^[0-9a-z]{100}$"},
		{"a", "", 3, "^a-[0-9a-z]{3}$"},
		{"-", "", 3, "^--[0-9a-z]{3}$"},
		{"", "b", 3, "^b-[0-9a-z]{3}$"},
		{"", "-", 3, "^--[0-9a-z]{3}$"},
		{"a", "b", 3, "^a-b-[0-9a-z]{3}$"},
		{"-", "-", 3, "^----[0-9a-z]{3}$"},

		{"prefix1", "customPart1", 15, "^prefix1-customPart1-[0-9a-z]{15}$"},
		{" \t\r\n", " \t\r\n", 3, "^ \t\r\n- \t\r\n-[0-9a-z]{3}$"},
	} {
		testName := strconv.Itoa(ti)
		t.Run(testName, func(t *testing.T) {
			// SETUP
			examinee := &namespaceManager{
				prefix:       tc.prefix,
				suffixLength: tc.suffixLength,
			}

			// EXERCISE
			result, err := examinee.generateName(tc.customPart)

			// VERIFY
			assert.NilError(t, err)
			assert.Assert(t, is.Regexp(tc.expectedResultRegexp, result))
		})
	}
}

func Test_namespaceManager_Create_uses_generateName(t *testing.T) {
	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory()
	examinee := &namespaceManager{
		nsInterface:  cf.CoreV1().Namespaces(),
		prefix:       "prefix1",
		suffixLength: 17,
	}

	// EXERCISE
	result, err := examinee.Create(ctx, "customPart1", map[string]string{})

	// VERIFY
	assert.NilError(t, err)
	assert.Assert(t, is.Regexp("^prefix1-customPart1-[0-9a-z]{17}$", result))
}

func Test_namespaceManager_Create_Success(t *testing.T) {
	// SETUP
	const namespaceName = "namespace1"

	ctx := context.Background()
	cf := fake.NewClientFactory(
	// no objects preexist
	)
	examinee := NewNamespaceManager(cf, "", 0)
	annotations := map[string]string{
		"key1":         "0439u5kfgn",
		"key2":         "9087652346",
		"04385340":     "0493785gns",
		"gbkjsn495678": "0948534etgdflk",
	}

	// EXERCISE
	result, err := examinee.Create(ctx, namespaceName, annotations)

	// VERIFY
	assert.NilError(t, err)
	assert.Equal(t, namespaceName, result)
	namespaceList, err := listNamespaces(ctx, cf)
	assert.NilError(t, err)
	assert.Equal(t, 1, len(namespaceList.Items))
	namespace := namespaceList.Items[0]
	assert.Equal(t, namespaceName, namespace.Name)
	assert.DeepEqual(t, annotations, namespace.GetObjectMeta().GetAnnotations())
}

func Test_namespaceManager_Create_ExistsAlready(t *testing.T) {
	// SETUP
	const namespaceName = "namespace1"

	ctx := context.Background()
	cf := fake.NewClientFactory(
		fake.Namespace(namespaceName), // existing namespace
	)
	examinee := NewNamespaceManager(cf, "", 0)

	// EXERCISE
	result, err := examinee.Create(ctx, namespaceName, map[string]string{})

	// VERIFY
	assert.Assert(t, err != nil)
	assert.Equal(t, "", result)
}

func Test_namespaceManager_Delete_Success(t *testing.T) {
	// SETUP
	const namespaceName = "namespace1"

	ctx := context.Background()
	cf := fake.NewClientFactory(
		fake.Namespace(namespaceName),
	)
	examinee := NewNamespaceManager(cf, "", 0)
	assert.Equal(t, 1, countNamespaces(ctx, cf))

	// EXERCISE
	err := examinee.Delete(ctx, namespaceName)

	// VERIFY
	assert.NilError(t, err)
	assert.Equal(t, 0, countNamespaces(ctx, cf))
}

func Test_namespaceManager_Delete_FailsIfNameDoesNotStartWithPrefix(t *testing.T) {
	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory()
	examinee := NewNamespaceManager(cf, "prefix1", 0)

	// EXERCISE
	err := examinee.Delete(ctx, "foo")

	// VERIFY
	assert.Assert(t, err != nil)
	assert.Equal(t, "refused to delete namespace 'foo': name does not start with 'prefix1'", err.Error())
}

func Test_namespaceManager_Delete_FailsIfPrefixLabelDoesNotMatch(t *testing.T) {
	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory()
	examinee := NewNamespaceManager(cf, "prefix1", 0)
	namespaceName, err := examinee.Create(ctx, "foo", map[string]string{})
	assert.NilError(t, err)

	namespace, err := cf.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	assert.NilError(t, err)
	labels := namespace.GetLabels()
	labels[labelPrefix] = "unexpectedValue"
	namespace.SetLabels(labels)
	cf.CoreV1().Namespaces().Update(ctx, namespace, metav1.UpdateOptions{})

	// EXERCISE
	err = examinee.Delete(ctx, namespaceName)

	// VERIFY
	assert.Assert(t, err != nil)
	assert.Equal(t, "refused to delete namespace 'prefix1-foo': not a Steward namespace (label mismatch)", err.Error())
}

func Test_namespaceManager_Delete_NotExisting(t *testing.T) {
	// SETUP
	ctx := context.Background()
	cf := fake.NewClientFactory(
	// no namespace preexists
	)
	examinee := NewNamespaceManager(cf, "", 0)
	assert.Equal(t, 0, countNamespaces(ctx, cf))

	// EXERCISE
	err := examinee.Delete(ctx, "foo")

	// VERIFY
	assert.NilError(t, err)
	assert.Equal(t, 0, countNamespaces(ctx, cf))
}

func listNamespaces(ctx context.Context, cf ClientFactory) (*corev1.NamespaceList, error) {
	return cf.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
}

func countNamespaces(ctx context.Context, factory ClientFactory) int {
	namespace, err := listNamespaces(ctx, factory)
	if err != nil {
		panic(err)
	}
	return len(namespace.Items)
}
