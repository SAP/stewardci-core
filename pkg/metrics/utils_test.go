package metrics

import (
	"testing"

	"gotest.tools/assert"
)

func Test_CodeLocation_WithoutSkip(t *testing.T) {
	// SETUP

	// EXERCISE
	var result string
	func() {
		result = CodeLocation(0)
	}()

	// VERIFY
	assert.Equal(t, result, "github.com/SAP/stewardci-core/pkg/metrics.Test_CodeLocation_WithoutSkip.func1")
}

func Test_CodeLocation_WithSkip(t *testing.T) {
	// SETUP

	// EXERCISE
	var result string
	func() {
		func() {
			func() {
				func() {
					result = CodeLocation(2)
				}()
			}()
		}()
	}()

	// VERIFY
	assert.Equal(t, result, "github.com/SAP/stewardci-core/pkg/metrics.Test_CodeLocation_WithSkip.func1.1")
}
