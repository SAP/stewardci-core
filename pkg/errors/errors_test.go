package errors

import (
	"fmt"
	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	"gotest.tools/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_ErrorMessage(t *testing.T) {
	err := Errorf(fmt.Errorf("%s", "unknown"), "%s", "bla")
	assert.Equal(t, "bla: unknown", err.Error())
}

func Test_ErrorWithUnknownStatus(t *testing.T) {
	err := Errorf(fmt.Errorf("%s", "unknown"), "%s", "bla")
	assert.Equal(t, metav1.StatusReasonUnknown, k8serrors.ReasonForError(err))
}

func Test_Error_IsNotFound(t *testing.T) {
	cause := k8serrors.NewNotFound(stewardv1alpha1.Resource("foo"), "myName")
	err := Errorf(cause, "%s", "Not found")
	assert.Assert(t, err.IsNotFound())
	assert.Equal(t, fmt.Sprintf("Not found: %s", cause.Error()), err.Error())
}
