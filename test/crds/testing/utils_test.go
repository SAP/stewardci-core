package testing

import (
	"testing"

	"gotest.tools/v3/assert"
)

func Test_FixIndent(t *testing.T) {
	in := "\t\t1\n\t\t\t2\n\t\t\t\t3\n\t\t\t\t\t4"
	out := FixIndent(in)
	assert.Equal(t, out, "1\n  2\n    3\n      4")
}
