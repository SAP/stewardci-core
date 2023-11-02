package custom

import (
	"strings"

	"github.com/lithammer/dedent"
)

// fixIndent removes common leading whitespace from all lines
// and replaces all tabs by spaces
func fixIndent(s string) (out string) {
	const TAB = "   "
	out = s
	out = dedent.Dedent(out)
	out = strings.ReplaceAll(out, "\t", TAB)
	return
}
