package k8s

import (
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_CopySecrets_NoFilter(t *testing.T) {
	namespace := ns1
	targetNamespace := "targetNs"
	cf := fake.NewClientFactory(fake.Secret("foo", namespace))
	tn := NewTenantNamespace(cf, namespace)
	targetClient := cf.CoreV1().Secrets(targetNamespace)
	helper := NewSecretHelper(tn, "", targetClient)

	list, err := helper.CopySecrets([]string{"foo"}, nil)
	assert.NilError(t, err)
	assert.Equal(t, "foo", list[0])
	storedSecret, _ := targetClient.Get("foo", metav1.GetOptions{})
	assert.Equal(t, "foo", storedSecret.GetName(), "Name should be equal")
}

func Test_CopySecrets_MapName(t *testing.T) {
	namespace := ns1
	targetNamespace := "targetNs"
	cf := fake.NewClientFactory(fake.Secret("foo", namespace))
	tn := NewTenantNamespace(cf, namespace)
	targetClient := cf.CoreV1().Secrets(targetNamespace)
	helper := NewSecretHelper(tn, "", targetClient)

	list, err := helper.CopySecrets([]string{"foo"}, nil, AppendNameSuffixFunc("suffix"))
	assert.NilError(t, err)
	assert.Equal(t, "foo-suffix", list[0])
	storedSecret, _ := targetClient.Get("foo-suffix", metav1.GetOptions{})
	assert.Equal(t, "foo-suffix", storedSecret.GetName(), "Name should be equal")
}

func Test_CopySecrets_DockerOnly(t *testing.T) {
	namespace := ns1
	targetNamespace := "targetNs"
	cf := fake.NewClientFactory(fake.Secret("foo", namespace),
		fake.SecretWithType("docker1", namespace, v1.SecretTypeDockercfg),
		fake.SecretWithType("docker2", namespace, v1.SecretTypeDockerConfigJson),
	)
	tn := NewTenantNamespace(cf, namespace)
	targetClient := cf.CoreV1().Secrets(targetNamespace)
	helper := NewSecretHelper(tn, "", targetClient)
	list, err := helper.CopySecrets([]string{"foo", "docker1", "docker2"}, DockerOnly)
	assert.NilError(t, err)
	assert.Equal(t, "docker1", list[0])
	assert.Equal(t, "docker2", list[1])
}

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
