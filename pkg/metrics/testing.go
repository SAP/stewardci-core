package metrics

import "github.com/prometheus/client_golang/prometheus"

// Testing provides utility functions for testing with this package.
// Do not use it for non-testing purposes!
type Testing struct{}

// PatchRegistry replaces the internal Prometheus metrics registry with a
// replacement and returns a function that reverts the patch.
// Multiple nested replacements must be reverted in exactly the opposite
// order (revert last replacement first).
func (Testing) PatchRegistry(replacement *prometheus.Registry) func() {
	origValue := registry
	registry = replacement
	return func() {
		if registry != replacement {
			panic("reverting not possible because current value is not the former replacement")
		}
		registry = origValue
	}
}
