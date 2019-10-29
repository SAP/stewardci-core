package secrets

import (
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	"testing"
)

func Test_AppendNameSuffixFunc(t *testing.T) {
	secret := fake.Secret("name", "secret")
	appendSuffix := AppendNameSuffixFunc("suffix")
	secret = appendSuffix(secret)
	assert.Equal(t, "name-suffix", secret.GetName())
}

func Test_SetAnnotationFunc_New(t *testing.T) {
	secret := fake.Secret("name", "secret")
	add := SetAnnotationFunc("foo", "bar")
	assert.Equal(t, "", secret.GetAnnotations()["foo"])
	secret = add(secret)
	assert.Equal(t, "bar", secret.GetAnnotations()["foo"])
}

func Test_SetAnnotationFunc_Overwrite(t *testing.T) {
	secret := fake.Secret("name", "secret")
	add := SetAnnotationFunc("foo", "bar")
	overwrite := SetAnnotationFunc("foo", "baz")
	secret = add(secret)
	assert.Equal(t, "bar", secret.GetAnnotations()["foo"])
	secret = overwrite(secret)
	assert.Equal(t, "baz", secret.GetAnnotations()["foo"])
}

func Test_StripAnnotationsFunc(t *testing.T) {
	secret := fake.Secret("name", "secret")
	add := SetAnnotationFunc("foo", "bar")
	strip := StripAnnotationsFunc("f")
	secret = add(secret)
	assert.Equal(t, "bar", secret.GetAnnotations()["foo"])
	secret = strip(secret)
	assert.Equal(t, "", secret.GetAnnotations()["foo"])
}

func Test_StripAnnotationsFunc_Empty(t *testing.T) {
	secret := fake.Secret("name", "secret")
	strip := StripAnnotationsFunc("f")
	secret = strip(secret)
	assert.Equal(t, "", secret.GetAnnotations()["foo"])
}

func Test_SetLabelFunc(t *testing.T) {
	secret := fake.Secret("name", "secret")
	add := SetLabelFunc("foo", "bar")
	overwrite := SetLabelFunc("foo", "baz")
	assert.Equal(t, "", secret.GetLabels()["foo"])
	secret = add(secret)
	assert.Equal(t, "bar", secret.GetLabels()["foo"])
	secret = overwrite(secret)
	assert.Equal(t, "baz", secret.GetLabels()["foo"])
}

func Test_StripLabelFunc(t *testing.T) {
	secret := fake.Secret("name", "secret")
	add := SetLabelFunc("foo", "bar")
	strip := StripLabelsFunc("f")
	secret = add(secret)
	assert.Equal(t, "bar", secret.GetLabels()["foo"])
	secret = strip(secret)
	assert.Equal(t, "", secret.GetLabels()["foo"])
}

func Test_StripLabelFunc_Empty(t *testing.T) {
	secret := fake.Secret("name", "secret")
	strip := StripLabelsFunc("f")
	secret = strip(secret)
	assert.Equal(t, "", secret.GetLabels()["foo"])
}
