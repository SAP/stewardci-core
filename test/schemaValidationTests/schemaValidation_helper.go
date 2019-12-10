package schemavalidationtests

import (
	"fmt"
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
func fixIndent(tabsToRemove int, s string, param string) (out string) {
	const TAB = "  "
	out = ""
	for _, line := range strings.Split(s, "\n") {
		out += strings.Replace(line, "\t", "", tabsToRemove) + "\n"
	}
	if param != "" {
		out = fmt.Sprintf(out, param)
	}
	out = strings.ReplaceAll(out, "\t", TAB)
	return
}
