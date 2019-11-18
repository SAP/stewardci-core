package fake

import (
	"fmt"
	"log"

	utils "github.com/SAP/stewardci-core/pkg/utils"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
)

// NewGenerateNameReactor returns a new ReactionFunc generating a name with
// generateName as prefix and a random with defined length as suffix
func NewGenerateNameReactor(randomLength int64) testing.ReactionFunc {
	return func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(testing.CreateActionImpl)
		accessor, err := meta.Accessor(createAction.Object)
		if err != nil {
			return false, nil, err
		}
		generateName := accessor.GetGenerateName()
		if generateName != "" {
			rand, _ := utils.RandomAlphaNumString(randomLength)
			accessor.SetName(fmt.Sprintf("%s%s", generateName, rand))
			accessor.SetClusterName(generateName)
		}
		log.Printf("Object: %+v", createAction.Object)
		return false, createAction.Object, nil
	}
}

// NewErrorReactor returns a new ReactorFunc returning an error
func NewErrorReactor(expectedErr error) testing.ReactionFunc {
	return func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, expectedErr
	}
}
