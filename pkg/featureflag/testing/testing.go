/*
Package testing provides utilities for tests that depend on
feature flags.

Feature flags are shared state. When tests change feature flags,
they must ensure that no other tests are affected.

 1. They must restore the default feature flag state when the
    test is finished.

 2. They must not run in parallel with other tests that could be
    affected by changed feature flags state.
*/
package testing

import "github.com/SAP/stewardci-core/pkg/featureflag"

/*
WithFeatureFlag sets the given feature flag to the given state and
returns a reset function to revert to the previous feature flag state.

The returned function is meant to be called deferred by the caller.

Example:

	defer testing.WithFeatureFlag(featureflag.Dummy, true)()
*/
func WithFeatureFlag(ff *featureflag.FeatureFlag, enabled bool) func() {
	changeFeatureFlag := func(ff *featureflag.FeatureFlag, enabled bool) {
		flagstr := ""
		if !enabled {
			flagstr = "-"
		}
		flagstr += ff.Key
		featureflag.ParseFlags(flagstr)
	}

	orig := ff.Enabled()
	changeFeatureFlag(ff, enabled)
	return func() {
		changeFeatureFlag(ff, orig)
	}
}
