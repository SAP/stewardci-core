package framework

import (
	"context"
	"testing"
	"time"

	"gotest.tools/assert"
)

func Test_Set_GetWaitInterval(t *testing.T) {
	// SETUP
	ctx := context.Background()
	interval := time.Duration(23)
	// EXERCISE
	ctx = SetWaitInterval(ctx, interval)
	result := GetWaitInterval(ctx)
	// VALIDATE
	assert.Equal(t, interval, result)
}

func Test_GetWaitInterval_return_default(t *testing.T) {
	// SETUP
	ctx := context.Background()
	// EXERCISE
	result := GetWaitInterval(ctx)
	// VALIDATE
	assert.Equal(t, defaultInterval, result)
}
