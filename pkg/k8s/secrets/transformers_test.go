package secrets

import (
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	"testing"
)

func Test_AppendNameSuffixTransformer(t *testing.T) {
	secret := fake.Secret("name", "secret")
	result := AppendNameSuffixTransformer("suffix")(secret)
	assert.Equal(t, "name", secret.GetName())
	assert.Equal(t, "name-suffix", result.GetName())
}

func Test_SetAnnotationTransformer_New(t *testing.T) {
	secret := fake.Secret("name", "secret")
	result := SetAnnotationTransformer("foo", "bar")(secret)
	assert.Equal(t, "", secret.GetAnnotations()["foo"])
	assert.Equal(t, "bar", result.GetAnnotations()["foo"])
}

func Test_SetAnnotationTransformer_Overwrite(t *testing.T) {
	secret := fake.Secret("name", "secret")
	result1 := SetAnnotationTransformer("foo", "bar")(secret)
	result2 := SetAnnotationTransformer("foo", "baz")(result1)
	assert.Equal(t, "bar", result1.GetAnnotations()["foo"])
	assert.Equal(t, "baz", result2.GetAnnotations()["foo"])
}

func Test_StripAnnotationsTransformer(t *testing.T) {
	secret := fake.Secret("name", "secret")
	result1 := SetAnnotationTransformer("foo", "bar")(secret)
	result2 := StripAnnotationsTransformer("f")(result1)
	assert.Equal(t, "bar", result1.GetAnnotations()["foo"])
	assert.Equal(t, "", result2.GetAnnotations()["foo"])
}

func Test_StripAnnotationsTransformer_Empty(t *testing.T) {
	secret := fake.Secret("name", "secret")
	result := StripAnnotationsTransformer("f")(secret)
	assert.Equal(t, "", result.GetAnnotations()["foo"])
}

func Test_SetLabelTransformer(t *testing.T) {
	secret := fake.Secret("name", "secret")
	result1 := SetLabelTransformer("foo", "bar")(secret)
	result2 := SetLabelTransformer("foo", "baz")(result1)
	assert.Equal(t, "", secret.GetLabels()["foo"])
	assert.Equal(t, "bar", result1.GetLabels()["foo"])
	assert.Equal(t, "baz", result2.GetLabels()["foo"])
}

func Test_StripLabelTransformer(t *testing.T) {
	secret := fake.Secret("name", "secret")
	result1 := SetLabelTransformer("foo", "bar")(secret)
	result2 := StripLabelsTransformer("f")(result1)
	assert.Equal(t, "bar", result1.GetLabels()["foo"])
	assert.Equal(t, "", result2.GetLabels()["foo"])
}

func Test_StripLabelTransformer_Empty(t *testing.T) {
	secret := fake.Secret("name", "secret")
	result := StripLabelsTransformer("f")(secret)
	assert.Equal(t, "", result.GetLabels()["foo"])
}
