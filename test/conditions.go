package test

import (
	"github.com/SAP/stewardci-core/pkg/k8s"
)

type WaitConditionFunc func(k8s.ClientFactory) (bool, error)

type waitCondition struct {
	conditionFunc WaitConditionFunc
	name          string
}

type WaitCondition interface {
	Wait(k8s.ClientFactory) (bool, error)
	Name() string
}

func NewWaitCondition(f WaitConditionFunc, name string) WaitCondition {
	return &waitCondition{
		conditionFunc: f,
		name:          name,
	}
}

func (w *waitCondition) Wait(clientFactory k8s.ClientFactory) (bool, error) {
	return w.conditionFunc(clientFactory)
}

func (w *waitCondition) Name() string {
	return w.name
}
