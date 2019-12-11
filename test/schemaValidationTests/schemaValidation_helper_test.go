package schemavalidationtests

import (
	"testing"

	"gotest.tools/assert"
)

func Test_fixIndent(t *testing.T) {
	in := "\t\t1\n\t\t\t2\n\t\t\t\t3\n\t\t\t\t\t4"
	out := fixIndent(2, in, "")
	assert.Equal(t, out, "1\n  2\n    3\n      4\n")
}

func Test_fixIndentWithParam(t *testing.T) {
	in := "\t\t1\n\t\t\t2\n\t\t\t\t3%v\n\t\t\t\t\t4"
	out := fixIndent(2, in, "xyz")
	assert.Equal(t, out, "1\n  2\n    3xyz\n      4\n")
}
