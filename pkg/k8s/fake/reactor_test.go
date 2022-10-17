package fake

import (
	"context"
	"strconv"
	"testing"

	assert "gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubernetes "k8s.io/client-go/kubernetes/fake"
	k8sclienttesting "k8s.io/client-go/testing"
)

func Test_GenerateNameReactor_GoodCases(t *testing.T) {
	const (
		randomSufficLen = 13
	)

	for ti, tc := range []struct {
		origName             string
		origGenerateName     string
		expectedNameRegEx    string
		expectedGenerateName string
	}{
		{"", "", "", ""},
		{"origName1", "", "origName1", ""},
		// existing name must not be changed:
		{"origName1", "origGenName1-", "origName1", "origGenName1-"},
		{"", "origGenName1-", "origGenName1-[0-9a-z]{13}", "origGenName1-"},
	} {
		t.Run(strconv.Itoa(ti), func(t *testing.T) {
			// SETUP
			origObj := v1.Namespace{ // just an example resource type
				ObjectMeta: metav1.ObjectMeta{
					Name:         tc.origName,
					GenerateName: tc.origGenerateName,
				},
			}
			action := k8sclienttesting.NewRootCreateAction(
				v1.SchemeGroupVersion.WithResource("namespaces"),
				origObj.DeepCopy(),
			)
			examinee := GenerateNameReactor(randomSufficLen)

			// EXERCISE
			resultHandled, resultObj, resultErr := examinee(action)

			// VERIFY
			assert.Assert(t, resultHandled == false)
			assert.Assert(t, resultObj != nil)
			assert.NilError(t, resultErr)

			assert.Assert(t, is.Regexp(tc.expectedNameRegEx, resultObj.(*v1.Namespace).Name))

			// no other fields modified
			{
				expectedObj := origObj.DeepCopy()
				expectedObj.SetName("") // exclude from comparison

				actualObj := resultObj.(*v1.Namespace).DeepCopy()
				actualObj.SetName("") // exclude from comparison

				assert.DeepEqual(t, expectedObj, actualObj)
			}
		})
	}
}

func Test_GenerateNameReactor_PanicsIfActionIsNotACreateAction(t *testing.T) {
	// SETUP
	action := k8sclienttesting.NewRootGetAction(
		v1.SchemeGroupVersion.WithResource("dummy"),
		"someResourceName",
	)
	examinee := GenerateNameReactor(13)

	// EXERCISE
	assert.Assert(t, is.Panics(func() {
		examinee(action)
	}))
}

func Test_CreationTimeReactor(t *testing.T) {
	// SETUP
	ctx := context.Background()
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
	factory := NewClientFactory()
	cs := factory.KubernetesClientset()
	cs.PrependReactor("create", "*", NewCreationTimestampReactor())
	client := cs.CoreV1().Namespaces()

	// EXERCISE
	client.Create(ctx, namespace, metav1.CreateOptions{})

	// VERIFY
	ns, err := client.Get(ctx, "foo", metav1.GetOptions{})
	assert.NilError(t, err)
	created := ns.GetCreationTimestamp()
	assert.Assert(t, false == (&created).IsZero())
}

type objectWithoutMetadataForTests struct{}

func (*objectWithoutMetadataForTests) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

func (*objectWithoutMetadataForTests) DeepCopyObject() runtime.Object {
	return &objectWithoutMetadataForTests{}
}

var _ runtime.Object = &objectWithoutMetadataForTests{}

func Test_GenerateNameReactor_PanicsIfObjectHasNoMetadata(t *testing.T) {
	// SETUP
	obj := &objectWithoutMetadataForTests{}
	action := k8sclienttesting.NewRootCreateAction(
		v1.SchemeGroupVersion.WithResource("dummy"),
		obj,
	)
	examinee := GenerateNameReactor(13)

	// EXERCISE
	assert.Assert(t, is.Panics(func() {
		examinee(action)
	}))
}

func Test_GenerateNameReactor_ExampleUsageOnFakeClientset(t *testing.T) {
	// SETUP
	ctx := context.Background()
	clientset := kubernetes.NewSimpleClientset()

	// attach reactor to fake clientset
	clientset.PrependReactor("create", "*", GenerateNameReactor(5))

	origObj := &v1.Namespace{ // just an example resource type
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "prefix1-",
		},
	}
	client := clientset.CoreV1().Namespaces()

	// EXERCISE
	resultObj, resultErr := client.Create(ctx, origObj, metav1.CreateOptions{})

	// VERIFY
	assert.NilError(t, resultErr)
	assert.Assert(t, is.Regexp(`^prefix1-[0-9a-z]{5}$`, resultObj.GetName()))

	storedObj, err := client.Get(ctx, resultObj.GetName(), metav1.GetOptions{})
	assert.NilError(t, err)

	assert.DeepEqual(t, resultObj, storedObj)
}
