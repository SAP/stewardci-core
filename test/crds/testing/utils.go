package testing

import (
	"fmt"
	"strings"

	"github.com/lithammer/dedent"
	"gotest.tools/v3/assert/cmp"
)

// FixIndent removes common leading whitespace from all lines
// and replaces all tabs by spaces
func FixIndent(s string) (out string) {
	const TAB = "  "
	out = s
	out = dedent.Dedent(out)
	out = strings.ReplaceAll(out, "\t", TAB)
	return
}

// ErrorContainsToken returns a comparison which checks whether
// the given error contains the given token.
// A token is contained only if there is a substring in the error
// message equal to the token, at the start and end of the substring
// there's a word boundary and the substring is not preceeded or
// followed by dot.
func ErrorContainsToken(err error, token string) cmp.Comparison {
	return cmp.Regexp(
		fmt.Sprintf(
			// token must start and end at a word boundary
			// token must not be preceeded or followed by dot
			`.*($|[^.])\b\Q%s\E\b($|[^.]).*`,
			strings.ReplaceAll(token, `\E`, `\E\\E\Q`),
		),
		err.Error(),
	)
}
