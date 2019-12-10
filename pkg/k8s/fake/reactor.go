package fake

import (
	utils "github.com/SAP/stewardci-core/pkg/utils"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
)

// GenerateNameReactor returns a new ReactionFunc simulating
// resource name generation via `metadata.generateName` as
// performed by a K8s API Server.
// If `metadata.name` is NOT set and `metadata.generateName` is
// set to a non-empty string, this reactor sets `metadata.name`
// to the value of `metadata.generateName` with a random alphanumeric
// suffix of the given length appended.
// Note that the returned reactor does not guarantee uniqueness -
// it might generate a name that is already used in the fake
// clientset.
func GenerateNameReactor(randomLength int64) testing.ReactionFunc {
	return func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(testing.CreateAction)
		accessor, err := meta.Accessor(createAction.GetObject())
		if err != nil {
			panic(err)
		}
		generateName := accessor.GetGenerateName()
		if accessor.GetName() == "" && generateName != "" {
			rand, err := utils.RandomAlphaNumString(randomLength)
			if err != nil {
				panic(err)
			}
			accessor.SetName(generateName + rand)
		}
		return false, createAction.GetObject(), nil
	}
}

// NewErrorReactor returns a new ReactorFunc returning an error
func NewErrorReactor(expectedErr error) testing.ReactionFunc {
	return func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, expectedErr
	}
}

// NewCreationTimestampReactor returns a new ReactorFunc setting the creation time
func NewCreationTimestampReactor() testing.ReactionFunc {
	return func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(testing.CreateAction)
		accessor, err := meta.Accessor(createAction.GetObject())
		if err != nil {
			panic(err)
		}
		accessor.SetCreationTimestamp(metav1.Now())
		return false, createAction.GetObject(), nil
	}
}
