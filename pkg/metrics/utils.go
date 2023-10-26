package metrics

import (
	"fmt"
	"runtime"
)

// CodeLocation returns a string representation of the caller's
// code location.
// `skip` is the number of call stack frames to skip.
func CodeLocation(skip uint16) string {
	skip += 2 // always skip this and `runtime.Callers` frame
	pc := make([]uintptr, 1)
	entryCount := runtime.Callers(int(skip), pc)
	if entryCount == 0 {
		panic(fmt.Errorf("cannot identify caller when skipping %d frames", skip))
	}
	frames := runtime.CallersFrames(pc)
	frame, _ := frames.Next()
	if frame.Function != "" {
		return frame.Function
	}
	return "<Unknown>"
}
