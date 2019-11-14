// +build experimental

package fake

import (
	"log"
	"testing"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes/fake"
)

func Test_GenerateNameReactor(t *testing.T) {
	t.Parallel()
	// SETUP
	randomLength := int64(3)
	clientset := kubernetes.NewSimpleClientset()
	clientset.PrependReactor("create", "*", NewGenerateNameReactor(randomLength))
	secretTemplate := &v1.Secret{ObjectMeta: metav1.ObjectMeta{
		GenerateName: "prefix1-",
		Name:         "prefix1-abc",
		Namespace:    "ns1",
	},
	}
	secretsClient := clientset.CoreV1().Secrets("ns1")
	// EXERCISE
	secret1, err := secretsClient.Create(secretTemplate)
	log.Printf("Secret1: %v", secret1)
	se, _ := secretsClient.Get("prefix1-abc", metav1.GetOptions{})
	log.Printf("Secret Stored: %v", se)
	//secret2, err := secretsClient.Create(secretTemplate)
	// VERIFY
	assert.NilError(t, err)
	assert.Assert(t, is.Regexp("prefix1-.*", secret1.GetName()))
	assert.Equal(t, "prefix1-", secret1.GetClusterName())
	//assert.Assert(t, is.Regexp("prefix1- .*", secret2.GetName()))
	//assert.Assert(t, !(secret1.GetName() == secret2.GetName()))
}
