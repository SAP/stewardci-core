package runctl

import (
	"fmt"
	assert "gotest.tools/assert"
	"testing"
)

func Test_IsRecoverable_recoverable_error_returns_true(t *testing.T) {
	err := NewRecoverabilityInfoError(fmt.Errorf("foo"), true)
	assert.Assert(t, IsRecoverable(err) == true)
}

func Test_IsRecoverable_permanent_error_returns_false(t *testing.T) {
	err := NewRecoverabilityInfoError(fmt.Errorf("foo"), false)
	assert.Assert(t, IsRecoverable(err) == false)
}

func Test_IsRecoverable_otherError_returns_false(t *testing.T) {
	t.Parallel()
	assert.Assert(t, IsRecoverable(fmt.Errorf("My Error")) == false)
}
