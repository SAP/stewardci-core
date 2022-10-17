package utils

import (
	"math"
	"strconv"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func Test_Random_Negative(t *testing.T) {
	random, err := RandomAlphaNumString(-1)
	assert.NilError(t, err)
	assert.Equal(t, "", random)
}

func Test_Random(t *testing.T) {
	const testRounds = 5000

	for _, length := range []int64{
		0,
		1,
		2,
		10,
		30,
		math.MaxUint8,
	} {
		testName := strconv.Itoa(int(length))
		t.Run(testName, func(t *testing.T) {
			for i := 0; i < testRounds; i++ {
				// EXERCISE
				result, err := RandomAlphaNumString(length)

				// VERIFY
				assert.NilError(t, err)
				assert.Equal(t, int(length), len(result))
				assert.Assert(t, is.Regexp("^[0-9a-z]*$", result))
			}
		})
	}
}
