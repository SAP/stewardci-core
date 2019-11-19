package test

import (
	"context"
)

// WaitConditionFunc is a function waiting for a condition
// return true,nil if condition is fullfilled
// return false,nil if condition may be fullfilled in the future
// returns nil,error if condition is not fullfilled
type WaitConditionFunc func(context.Context) (bool, error)

type waitCondition struct {
	conditionFunc WaitConditionFunc
}

// WaitCondition interface implenments a Wait function
type WaitCondition interface {
	Check(context.Context) (bool, error)
}

// NewWaitCondition creates a new WaitCondition
// name must be unique
func NewWaitCondition(f WaitConditionFunc) WaitCondition {
	return &waitCondition{
		conditionFunc: f,
	}
}

// Check is checking for a condition
// return true,nil if condition is fullfilled
// return false,nil if condition may be fullfilled in the future
// returns nil,error if condition is not fullfilled
func (w *waitCondition) Check(ctx context.Context) (bool, error) {
	return w.conditionFunc(ctx)
}

