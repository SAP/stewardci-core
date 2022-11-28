/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package featureflag implements simple feature-flagging.
Feature flags can become an anti-pattern if abused.
We should try to use them for two use-cases:
  - `Preview` feature flags enable a piece of functionality we haven't yet fully baked.  The user needs to 'opt-in'.
    We expect these flags to be removed at some time.  Normally these will default to false.
  - Escape-hatch feature flags turn off a default that we consider risky (e.g. pre-creating DNS records).
    This lets us ship a behaviour, and if we encounter unusual circumstances in the field, we can
    allow the user to turn the behaviour off.  Normally these will default to true.

Feature flags are set via a single environment variable.
The value is a string of feature flag keys separated by sequences of
whitespace and comma.
Each key can be prefixed with `+` (enable flag) or `-` (disable flag).
Without prefix the flag gets enabled.
*/
package featureflag

import (
	"os"
	"regexp"
	"sync"

	"k8s.io/klog/v2"
)

const (
	// Name is the name of the environment variable which encapsulates feature flags.
	Name = "STEWARD_FEATURE_FLAGS"
)

func init() {
	ParseFlags(os.Getenv(Name))
}

var (
	flags      = make(map[string]*FeatureFlag)
	flagsMutex sync.Mutex
)

// FeatureFlag defines a feature flag
type FeatureFlag struct {
	Key          string
	enabled      *bool
	defaultValue *bool
}

// New creates a new feature flag.
func New(key string, defaultValue *bool) *FeatureFlag {
	flagsMutex.Lock()
	defer flagsMutex.Unlock()

	f := flags[key]
	if f == nil {
		f = &FeatureFlag{
			Key: key,
		}
		flags[key] = f
	}

	if f.defaultValue == nil {
		f.defaultValue = defaultValue
	}

	return f
}

// Enabled checks if the flag is enabled.
func (f *FeatureFlag) Enabled() bool {
	if f.enabled != nil {
		return *f.enabled
	}
	if f.defaultValue != nil {
		return *f.defaultValue
	}
	return false
}

// Bool returns a pointer to the boolean value.
func Bool(b bool) *bool {
	return &b
}

// ParseFlags is responsible for parse out the feature flag usage.
func ParseFlags(f string) {
	if f == "" {
		return
	}
	for _, s := range regexp.MustCompile(`[[:space:],]+`).Split(f, -1) {
		if s == "" {
			continue
		}
		enabled := true
		var ff *FeatureFlag
		if s[0] == '+' || s[0] == '-' {
			ff = New(s[1:], nil)
			if s[0] == '-' {
				enabled = false
			}
		} else {
			ff = New(s, nil)
		}
		klog.InfoS("feature flag", "key", ff.Key, "enabled", enabled)
		ff.enabled = &enabled
	}
}
