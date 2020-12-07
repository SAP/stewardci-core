package errors

import (
	"fmt"
	"testing"

	"gotest.tools/assert"

	api "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
)

func Test_GetClass_UnclassifiedError(t *testing.T) {
	err1 := fmt.Errorf("err1")

	assert.Equal(t, api.ResultUndefined, GetClass(err1))
}

func Test_Classify(t *testing.T) {
	err1 := fmt.Errorf("err1")

	for _, tc := range []struct {
		class api.Result
	}{
		{api.ResultErrorInfra},
		{api.ResultErrorContent},
		{api.ResultErrorConfig},
	} {
		t.Run(string(tc.class), func(t *testing.T) {

			//EXERCISE
			classifiedErr := Classify(err1, tc.class)

			// VERIFY
			assert.Equal(t, tc.class, GetClass(classifiedErr))
		})
	}
}
