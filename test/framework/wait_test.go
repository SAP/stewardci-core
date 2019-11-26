package framework

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

var waitFuncTrue WaitConditionFunc = func(context.Context) (bool, error) {
	return true, nil
}

var waitFuncFalse WaitConditionFunc = func(context.Context) (bool, error) {
	return false, nil
}

func waitFuncError(err error) WaitConditionFunc {
	return func(context.Context) (bool, error) {
		return true, err
	}
}

func waitFuncIntervals(x int) WaitConditionFunc {
	counter := 0
	return func(ctx context.Context) (bool, error) {
		time.Sleep(GetWaitInterval(ctx))
		counter++
		if counter >= x {
			return true, nil
		}
		return false, nil

	}
}

func Test_WaitFor_success(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name          string
		timeout       int // set to negative value for no timeout, set to 0 for immediate timout
		waitForFunc   WaitConditionFunc
		expectedError string
	}{{
		name:          "ok",
		timeout:       -1,
		waitForFunc:   waitFuncTrue,
		expectedError: "",
	}, {
		name:          "error",
		timeout:       -1,
		waitForFunc:   waitFuncError(fmt.Errorf("foo")),
		expectedError: "foo",
	}, {
		name:          "timeout_0x",
		timeout:       0,
		waitForFunc:   waitFuncFalse,
		expectedError: "context deadline exceeded",
	}, {
		name:          "timeout_1x",
		timeout:       1,
		waitForFunc:   waitFuncFalse,
		expectedError: "context deadline exceeded",
	}, {
		name:          "timeout_2x",
		timeout:       2,
		waitForFunc:   waitFuncFalse,
		expectedError: "context deadline exceeded",
	}, {
		name:          "sleep_3_timeout_2",
		timeout:       2,
		waitForFunc:   waitFuncIntervals(3),
		expectedError: "context deadline exceeded",
	}, {
		name:          "sleep_2_timeout_3",
		timeout:       3,
		waitForFunc:   waitFuncIntervals(2),
		expectedError: "",
	},
	} {
		t.Run(test.name, func(t *testing.T) {
			// SETUP
			test := test
			t.Parallel()
			ctx := context.Background()
			ctx = SetWaitInterval(ctx, time.Duration(100)*time.Millisecond)
			ctx = SetTestName(ctx, test.name)
			if test.timeout >= 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(test.timeout)*GetWaitInterval(ctx))
				defer cancel()
			}
			// EXERCISE
			_, result := WaitFor(ctx, test.waitForFunc)
			// VERIFY
			if test.expectedError == "" {
				assert.NilError(t, result)
			} else {
				assert.Assert(t, result != nil)
				assert.Assert(t, is.Regexp(test.expectedError, result.Error()))
			}
		})
	}
}
