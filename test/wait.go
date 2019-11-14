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

type Waiter interface {
	WaitFor(condition WaitCondition) error
}

type waiter struct {
	clientFactory k8s.ClientFactory
}

func NewWaiter(clientFactory k8s.ClientFactory) Waiter {
	return &waiter{clientFactory: clientFactory}
}

func (w *waiter) WaitFor(condition WaitCondition) error {
	_, span := trace.StartSpan(context.Background(), condition.Name())
	defer span.End()
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		return condition.Wait(w.clientFactory)
	})
}

func (w *waiter) MyWaitFor(condition WaitCondition) error {
	time.Sleep(interval)
	for {
		result, err := condition.Wait(w.clientFactory)
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
