package schemavalidationtests

import (
	"github.com/lithammer/dedent"
	"strings"
	"testing"
)

// SchemaValidationTest is a test for schema validation
type SchemaValidationTest struct {
	name       string
	data       string
	dataFormat format
	check      func(t *testing.T, err error)
}

type format string

const (
	json format = "JSON"
	yaml format = "YAML"
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
