package utils

import (
	"testing"

	"gotest.tools/assert"
)

func Test_Random(t *testing.T) {
	random, err := RandomAlphaNumString(2)
	assert.NilError(t, err)
	assert.Equal(t, 2, len(random))
}

func Test_Random_Empty(t *testing.T) {
	random, err := RandomAlphaNumString(0)
	assert.NilError(t, err)
	assert.Equal(t, "", random)
}

func Test_Random_Negative(t *testing.T) {
	random, err := RandomAlphaNumString(-1)
	assert.NilError(t, err)
	assert.Equal(t, "", random)
}
