package test

import (
	"context"
	"log"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	interval = 1 * time.Second
	timeout  = 2 * time.Minute
)

// WaitConditionFunc is a function waiting for a condition
// return true,nil if condition is fullfilled
// return false,nil if condition may be fullfilled in the future
// returns nil,error if condition is not fullfilled
type WaitConditionFunc func(context.Context) (bool, error)

// WaitFor waits for a condition
// it returns an error if condition is not fullfilled
func WaitFor(ctx context.Context, conditionFunc WaitConditionFunc) error {
	startTime := time.Now()
	log.Printf("wait for %s", GetTestName(ctx))
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		return conditionFunc(ctx)
	})
	log.Printf("waiting completed for %s after %s", GetTestName(ctx), time.Now().Sub(startTime))
	return err
}
