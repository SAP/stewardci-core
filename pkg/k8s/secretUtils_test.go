package k8s

import (
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
)

func Test_Rename(t *testing.T) {
	secret := fake.Secret("name", "secret")
	rename := Rename("newName")
	secret = rename(secret)
	assert.Equal(t, "newName", secret.GetName())
}

func Test_SetAnnotation_New(t *testing.T) {
	secret := fake.Secret("name", "secret")
	add := SetAnnotation("foo", "bar")
	assert.Equal(t, "", secret.GetAnnotations()["foo"])
	secret = add(secret)
	assert.Equal(t, "bar", secret.GetAnnotations()["foo"])
}

func Test_SetAnnotation_Overwrite(t *testing.T) {
	secret := fake.Secret("name", "secret")
	add := SetAnnotation("foo", "bar")
	overwrite := SetAnnotation("foo", "baz")
	secret = add(secret)
	assert.Equal(t, "bar", secret.GetAnnotations()["foo"])
	secret = overwrite(secret)
	assert.Equal(t, "baz", secret.GetAnnotations()["foo"])
}

func Test_StripAnnotations(t *testing.T) {
	secret := fake.Secret("name", "secret")
	add := SetAnnotation("foo", "bar")
	strip := StripAnnotations("f")
	secret = add(secret)
	assert.Equal(t, "bar", secret.GetAnnotations()["foo"])
	secret = strip(secret)
	assert.Equal(t, "", secret.GetAnnotations()["foo"])
}

func Test_StripAnnotations_Empty(t *testing.T) {
	secret := fake.Secret("name", "secret")
	strip := StripAnnotations("f")
	secret = strip(secret)
	assert.Equal(t, "", secret.GetAnnotations()["foo"])
}

func Test_SetLabel(t *testing.T) {
	secret := fake.Secret("name", "secret")
	add := SetLabel("foo", "bar")
	overwrite := SetLabel("foo", "baz")
	assert.Equal(t, "", secret.GetLabels()["foo"])
	secret = add(secret)
	assert.Equal(t, "bar", secret.GetLabels()["foo"])
	secret = overwrite(secret)
	assert.Equal(t, "baz", secret.GetLabels()["foo"])
}

func Test_StripLabel(t *testing.T) {
	secret := fake.Secret("name", "secret")
	add := SetLabel("foo", "bar")
	strip := StripLabels("f")
	secret = add(secret)
	assert.Equal(t, "bar", secret.GetLabels()["foo"])
	secret = strip(secret)
	assert.Equal(t, "", secret.GetLabels()["foo"])
}

func Test_StripLabel_Empty(t *testing.T) {
	secret := fake.Secret("name", "secret")
	strip := StripLabels("f")
	secret = strip(secret)
	assert.Equal(t, "", secret.GetLabels()["foo"])
}
