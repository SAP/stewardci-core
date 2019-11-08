package test

import (
	"context"
	"time"

	"go.opencensus.io/trace"
	"k8s.io/apimachinery/pkg/util/wait"

"github.com/SAP/stewardci-core/pkg/k8s"
)

const (
	interval = 1 * time.Second
	timeout  = 2 * time.Minute
)

func WaitForState(clientFactory k8s.ClientFactory, condition WaitCondition, name string) error {
	_, span := trace.StartSpan(context.Background(), name)
	defer span.End()

	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		return condition(clientFactory)
	})
}
