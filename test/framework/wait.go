package framework

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// WaitConditionFunc is a function waiting for a condition
// return true,nil if condition is fullfilled
// return false,nil if condition may be fullfilled in the future
// returns nil,error if condition is not fullfilled
type WaitConditionFunc func(context.Context) (bool, error)

// WaitFor waits for a condition
// it returns the duration the waiting took
// it returns an error if condition cannot be fullfilled anymore
func WaitFor(ctx context.Context, conditionFunc WaitConditionFunc) (time.Duration, error) {
	startTime := time.Now()
	err := wait.PollImmediateInfinite(GetWaitInterval(ctx), func() (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		return conditionFunc(ctx)
	})
	return time.Now().Sub(startTime), err
}
