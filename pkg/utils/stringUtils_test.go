package k8s

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)

func Test_Trim(t *testing.T) {
	result := Trim(" \t \r \n  abc \t \n \r ")
	assert.Equal(t, "abc", result)
}

func Test_ShortenMessage_shortened(t *testing.T) {
	result := ShortenMessage("  ABC\nDEF\r\r\nGHI \t ", 8)
	assert.Equal(t, "ABC D...", result)
}

func Test_ShortenMessage_tooShortLength(t *testing.T) {
	result := ShortenMessage("ABCDEF", 2)
	assert.Equal(t, "...", result)
}

func Test_ShortenMessage_negativeLength(t *testing.T) {
	result := ShortenMessage("ABCDEF", -5)
	assert.Equal(t, "...", result)
}

func Test_ShortenMessage_shortEnough_notCut(t *testing.T) {
	result := ShortenMessage("  ABC\nDEF\r\r\nGHI \t ", 12)
	assert.Equal(t, "ABC DEF GHI", result)
}

func Test_ShortenMessage_lineBreak_N(t *testing.T) {
	result := ShortenMessage(" A\nB ", 1000)
	assert.Equal(t, "A B", result)
}

func Test_ShortenMessage_lineBreak_R(t *testing.T) {
	result := ShortenMessage(" A\rB ", 1000)
	assert.Equal(t, "A B", result)
}

func Test_ShortenMessage_lineBreak_NR(t *testing.T) {
	result := ShortenMessage(" A\n\rB ", 1000)
	assert.Equal(t, "A B", result)
}

func Test_ShortenMessage_multipleBlanksSquashed(t *testing.T) {
	result := ShortenMessage(" A    B ", 1000)
	assert.Equal(t, "'A B'", fmt.Sprintf("'%s'", result))
}

func Test_ShortenMessage_MixedSquashed(t *testing.T) {
	result := ShortenMessage(" A  \t \n\r \n B ", 1000)
	assert.Equal(t, "'A B'", fmt.Sprintf("'%s'", result))
}

func Test_ShortenMessage_multipleNewLinesSquashed(t *testing.T) {
	result := ShortenMessage(" A\n\n\nB ", 1000)
	assert.Equal(t, "A B", result)
}
