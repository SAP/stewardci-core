package framework

import (
	"fmt"
	"testing"
"log"

)


// TODO Test negative tests
func Test_CheckResult(t *testing.T) {
log.Printf("%+v",t)
	t.Parallel()
	for _, test := range []struct {
		run      testRun
		expected bool
	}{
		{testRun{Expected: "", result: nil}, true},
		{testRun{Expected: "foo", result: fmt.Errorf("foo")}, true},
	} {
		// EXERCISE
		checkResult(t, test.run)

	}
}
