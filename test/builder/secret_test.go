package builder

import (
	"testing"

	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_SecretBasicAuth(t *testing.T) {
	secret := SecretBasicAuth("foo", "bar", "baz", "secret1")
	expectedsecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
			Labels:    map[string]string{"jenkins.io/credentials-type": "usernamePassword"},
		},
		Type: v1.SecretTypeOpaque,
		StringData: map[string]string{"username": "baz",
			"password": "secret1",
		},
	}

	assert.DeepEqual(t, expectedsecret, secret)
}
