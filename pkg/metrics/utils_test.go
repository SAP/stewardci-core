package metrics

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_CodeLocation_WithoutSkip(t *testing.T) {
	// SETUP
	g := NewGomegaWithT(t)

	// EXERCISE
	var result string
	func() {
		result = CodeLocation(0)
	}()

	// VERIFY
	g.Expect(result).To(Equal("github.com/SAP/stewardci-core/pkg/metrics.Test_CodeLocation_WithoutSkip.func1"))
}

func Test_CodeLocation_WithSkip(t *testing.T) {
	// SETUP
	g := NewGomegaWithT(t)

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
	// Inlining leads to different function names, see https://github.com/golang/go/issues/60324
	g.Expect(result).To(SatisfyAny(
		// with Go 1.18
		Equal("github.com/SAP/stewardci-core/pkg/metrics.Test_CodeLocation_WithSkip.func1.1"),

		// with Go 1.21
		Equal("github.com/SAP/stewardci-core/pkg/metrics.Test_CodeLocation_WithSkip.Test_CodeLocation_WithSkip.func1.func2"),
	))
}
