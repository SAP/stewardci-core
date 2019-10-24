package utils

import (
	"gotest.tools/assert"
	"testing"
)

func Test_Random(t *testing.T) {
	random, err := Random(2)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(random))
}

func Test_Random_Empty(t *testing.T) {
	random, err := Random(0)
	assert.NilError(t, err)
	assert.Equal(t, "", random)
}
