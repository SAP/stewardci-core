package testing

import "github.com/SAP/stewardci-core/pkg/runctl/metrics"

// PatchPipelineRunsPeriodic patches
// "github.com/SAP/stewardci-core/pkg/runctl/metrics".PipelineRunsPeriodic with
// the given replacement and returns a function that reverts the patch.
// Multiple nested replacements must be reverted in exactly the opposite order
// (revert last replacement first).
func PatchPipelineRunsPeriodic(replacement metrics.PipelineRunsMetric) func() {
	origValue := metrics.PipelineRunsPeriodic
	metrics.PipelineRunsPeriodic = replacement
	return func() {
		if metrics.PipelineRunsPeriodic != replacement {
			panic("reverting not possible because current value is not the former replacement")
		}
		metrics.PipelineRunsPeriodic = origValue
	}
}
