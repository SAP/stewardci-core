package runctl

import (
	"fmt"
	assert "gotest.tools/assert"
	"testing"
)

func Test_IsRecoverable(t *testing.T) {
	for _, test := range []struct {
		name     string
		err      error
		expected bool
	}{

		{name: "nil returns false",
			err:      nil,
			expected: false,
		},
		{name: "other error returns false",
			err:      fmt.Errorf("foo"),
			expected: false,
		},
		{name: "permanent error returns false",
			err:      NewRecoverabilityInfoError(fmt.Errorf("foo"), false),
			expected: false,
		},
		{name: "recoverable error returns true",
			err:      NewRecoverabilityInfoError(fmt.Errorf("foo"), true),
			expected: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			test := test
			t.Parallel()
			assert.Assert(t, test.expected == IsRecoverable(test.err))
		})
	}
}
