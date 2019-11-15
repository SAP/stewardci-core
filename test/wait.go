package test

import (
	"context"
	"log"
	"time"

	"github.com/SAP/stewardci-core/pkg/k8s"
	"go.opencensus.io/trace"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	interval = 1 * time.Second
	timeout  = 2 * time.Minute
)

// Waiter is a waiter waiting for a condition to be fullfilled
type Waiter interface {
	WaitFor(condition WaitCondition) error
}

type waiter struct {
	clientFactory k8s.ClientFactory
}

// NewWaiter returns a new Waiter
func NewWaiter(clientFactory k8s.ClientFactory) Waiter {
	return &waiter{clientFactory: clientFactory}
}

// WaitFor waits for a condition
// it returns an error if condition is not fullfilled
func (w *waiter) WaitFor(condition WaitCondition) error {
	log.Printf("wait for %s", condition.Name())
	startTime := time.Now()
	_, span := trace.StartSpan(context.Background(), condition.Name())
	defer span.End()

	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		return condition.Check(w.clientFactory)
	})
	log.Printf("waiting completed for %s after %s", condition.Name(), time.Now().Sub(startTime))
	return err
}

func (w *waiter) myWaitFor(condition WaitCondition) error {
	time.Sleep(interval)
	for {
		result, err := condition.Check(w.clientFactory)
		log.Printf("MyWaitFor: %t, %s", result, err)
		if err != nil {
			return err
		}
		if result {
			break
		}
		time.Sleep(interval)
	}
	return nil
}
