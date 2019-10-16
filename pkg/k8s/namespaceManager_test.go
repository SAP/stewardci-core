package k8s

import (
	"fmt"
	"strings"
	"testing"

	"log"

	stewardapi "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	emptyPrefix    = ""
	prefix         = "stu-test"
	preprefix      = "stu"
	noRandom       = 0
	shortRandom    = 5
	longRandom     = 16
	negativeRandom = -1

	nameSuffix        = "hello"
	nameSuffixEmpty   = ""
	nameSuffixIllegal = "hello_world"
)

var factory ClientFactory
var nsManager NamespaceManager

func Test_CreateNamespace_prefixSuffixShortRandom(t *testing.T) {
	setup(prefix, shortRandom)
	result, _ := create(nameSuffix)
	expectedPrefix := "stu-test-hello-"
	assert.Assert(t, strings.HasPrefix(result, expectedPrefix), "Unexpected: "+result)
	assert.Equal(t, expectedLength(expectedPrefix, shortRandom), len(result), "Unexpected: "+result)
}

func Test_CreateNamespace_prefixEmptySuffixLongRandom(t *testing.T) {
	setup(prefix, longRandom)
	result, _ := create(nameSuffixEmpty)
	expectedPrefix := "stu-test-"
	assert.Assert(t, strings.HasPrefix(result, expectedPrefix), "Unexpected: "+result)
	assert.Equal(t, expectedLength(expectedPrefix, longRandom), len(result), "Unexpected: "+result)
}

func Test_CreateNamespace_noRandom(t *testing.T) {
	setup(prefix, noRandom)
	result, _ := create(nameSuffix)
	expectedPrefix := "stu-test-hello"
	assert.Equal(t, expectedPrefix, result, "Unexpected: "+result)
}

func Test_CreateNamespace_negativeRandom(t *testing.T) {
	setup(prefix, negativeRandom)
	_, err := create(nameSuffix)
	assert.Assert(t, err != nil)
	assert.Equal(t, "randomLength not configured in namespace manager", err.Error())
}

func Test_CreateNamespace_alreadyExists(t *testing.T) {
	setup(prefix, noRandom)
	create(nameSuffix)
	result, _ := create(nameSuffix)
	assert.Equal(t, "", result, "Unexpected: Namespace created twice: "+result)
}

func Test_CreateDeleteNamespace(t *testing.T) {
	setup(prefix, longRandom)
	result, _ := create(nameSuffixEmpty)
	assert.Equal(t, 1, countNamespace(factory))
	assert.NilError(t, nsManager.Delete(result))
	assert.Equal(t, 0, countNamespace(factory))
}

func Test_DeleteNamespaceWithWrongManager_fails(t *testing.T) {
	setup(prefix, longRandom)
	result, _ := create(nameSuffixEmpty)
	nsManager2 := NewNamespaceManager(factory, preprefix, longRandom)

	namespaces := factory.CoreV1().Namespaces()
	namespace, _ := namespaces.List(metav1.ListOptions{})
	log.Printf("%+v", namespace)

	err := nsManager2.Delete(result)
	assert.Assert(t, err != nil)
	assert.Equal(t, "Cannot delete namespace not owned by this steward instance: '"+result+"'", err.Error())
}

func Test_DeleteNamespace_works(t *testing.T) {
	setup(prefix, noRandom)
	result, _ := create(nameSuffixEmpty)
	err := nsManager.Delete(result)
	assert.NilError(t, err)
	assert.Equal(t, 0, countNamespace(factory))
}

func Test_DeleteNamespaceWithWrongPrefix_fails(t *testing.T) {
	setup(prefix, longRandom)
	create(nameSuffixEmpty)

	nsName := "Wrong"
	err := nsManager.Delete(nsName)
	assert.Assert(t, err != nil)
	assert.Equal(t, fmt.Sprintf("Cannot delete namespace '%s'. It does not start with prefix '%s'", nsName, prefix), err.Error())
}

func Test_DeleteNamespaceNotExisting_success(t *testing.T) {
	setup(prefix, longRandom)
	nsName := prefix + "-NotExisting"
	err := nsManager.Delete(nsName)
	assert.NilError(t, err)
}

func expectedLength(expectedPrefix string, randomLength int) int {
	return len(expectedPrefix) + randomLength*2 //HEX
}

func setup(prefix string, random int) {
	factory = fake.NewClientFactory(
		fake.PipelineRun(run1, ns1, stewardapi.PipelineSpec{}),
	)
	nsManager = NewNamespaceManager(factory, prefix, random)
}

func create(nameSuffix string) (string, error) {
	name, err := nsManager.Create(nameSuffix, map[string]string{})
	return name, err
}

func countNamespace(factory ClientFactory) int {
	nf := factory.CoreV1().Namespaces()
	namespace, err := nf.List(metav1.ListOptions{})
	if err != nil {
		return -1
	}
	return len(namespace.Items)
}
