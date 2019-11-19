package secrets

import (
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
)

func Test_UniqueNameTransformer(t *testing.T) {
	secret := fake.SecretWithType("name1", "secret1", v1.SecretTypeDockercfg)
	result := UniqueNameTransformer()(secret)
	assert.Equal(t, "", result.GetName())
	assert.Equal(t, "name1-", result.GetGenerateName())
	assert.Equal(t, v1.SecretTypeDockercfg, result.Type)
}

func Test_SetAnnotationTransformer_New(t *testing.T) {
	secret := fake.SecretOpaque("name1", "secret1")
	result := SetAnnotationTransformer("foo", "bar")(secret)
	assert.Equal(t, "", secret.GetAnnotations()["foo"])
	assert.Equal(t, "bar", result.GetAnnotations()["foo"])
}

func Test_SetAnnotationTransformer_Overwrite(t *testing.T) {
	// SETUP
	secret := fake.SecretOpaque("name1", "secret1")
	secret = SetAnnotationTransformer("foo", "bar")(secret)
	assert.Equal(t, "bar", secret.GetAnnotations()["foo"])
	// EXERCISE
	result := SetAnnotationTransformer("foo", "baz")(secret)
	// VERIFY
	assert.Equal(t, "bar", secret.GetAnnotations()["foo"])
	assert.Equal(t, "baz", result.GetAnnotations()["foo"])
}

func Test_StripAnnotationsTransformer_match(t *testing.T) {
	// SETUP
	secret := fake.SecretOpaque("name1", "secret1")
	secret = SetAnnotationTransformer("foo", "bar")(secret)
	assert.Equal(t, "bar", secret.GetAnnotations()["foo"])
	// EXERCISE
	result := StripAnnotationsTransformer("f")(secret)
	// VERIFY
	assert.Equal(t, "bar", secret.GetAnnotations()["foo"])
	assert.Equal(t, "", result.GetAnnotations()["foo"])
}

func Test_StripAnnotationsTransformer_noMatch(t *testing.T) {
	// SETUP
	secret := fake.SecretOpaque("name1", "secret1")
	secret = SetAnnotationTransformer("foo", "bar")(secret)
	assert.Equal(t, "bar", secret.GetAnnotations()["foo"])
	// EXERCISE
	result := StripAnnotationsTransformer("x")(secret)
	// VERIFY
	assert.Equal(t, "bar", secret.GetAnnotations()["foo"])
	assert.Equal(t, "bar", result.GetAnnotations()["foo"])
}

func Test_StripAnnotationsTransformer_Empty(t *testing.T) {
	secret := fake.SecretOpaque("name1", "secret1")
	result := StripAnnotationsTransformer("f")(secret)
	assert.Equal(t, "", result.GetAnnotations()["foo"])
}

func Test_SetLabelTransformer(t *testing.T) {
	// SETUP
	secret := fake.SecretOpaque("name1", "secret1")
	secret = SetLabelTransformer("foo", "bar")(secret)
	assert.Equal(t, "bar", secret.GetLabels()["foo"])
	// EXERCISE
	result := SetLabelTransformer("foo", "baz")(secret)
	// VERIFY
	assert.Equal(t, "bar", secret.GetLabels()["foo"])
	assert.Equal(t, "baz", result.GetLabels()["foo"])
}

func Test_StripLabelTransformer_match(t *testing.T) {
	// SETUP
	secret := fake.SecretOpaque("name1", "secret1")
	secret = SetLabelTransformer("foo", "bar")(secret)
	assert.Equal(t, "bar", secret.GetLabels()["foo"])
	// EXERCISE
	result := StripLabelsTransformer("f")(secret)
	// VERIFY
	assert.Equal(t, "bar", secret.GetLabels()["foo"])
	assert.Equal(t, "", result.GetLabels()["foo"])
}

func Test_StripLabelTransformer_noMatch(t *testing.T) {
	// SETUP
	secret := fake.SecretOpaque("name1", "secret1")
	secret = SetLabelTransformer("foo", "bar")(secret)
	assert.Equal(t, "bar", secret.GetLabels()["foo"])
	// EXERCISE
	result := StripLabelsTransformer("x")(secret)
	// VERIFY
	assert.Equal(t, "bar", secret.GetLabels()["foo"])
	assert.Equal(t, "bar", result.GetLabels()["foo"])
}

func Test_StripLabelTransformer_Empty(t *testing.T) {
	secret := fake.SecretOpaque("name1", "secret1")
	result := StripLabelsTransformer("f")(secret)
	assert.Equal(t, "", result.GetLabels()["foo"])
}
