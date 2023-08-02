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

package featureflag

import (
	"os"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	"k8s.io/klog/v2/ktesting"
)

func TestFlagToFalse(t *testing.T) {
	f := New("UnitTest1", Bool(true))
	if !f.Enabled() {
		t.Fatalf("Flag did not default true")
	}

	ParseFlags("-UnitTest1")
	if f.Enabled() {
		t.Fatalf("Flag did not default turn off")
	}

	ParseFlags("UnitTest1")
	if !f.Enabled() {
		t.Fatalf("Flag did not default turn on")
	}
}

func TestSetenv(t *testing.T) {
	f := New("UnitTest2", Bool(true))
	if !f.Enabled() {
		t.Fatalf("Flag did not default true")
	}

	os.Setenv("STEWARD_FEATURE_FLAGS", "-UnitTest2")
	if !f.Enabled() {
		t.Fatalf("Flag was reparsed immediately after os.Setenv")
	}

	ParseFlags("-UnitTest2")
	if f.Enabled() {
		t.Fatalf("Flag was not updated by ParseFlags")
	}
}

func Test_Log(t *testing.T) {
	// SETUP
	g := NewGomegaWithT(t)

	origFlags := flags
	t.Cleanup(func() { flags = origFlags })
	flags = make(map[string]*FeatureFlag)

	// defined in non-lexical order
	New("a567943574457334", Bool(false))
	New("d896063233040385", Bool(true))
	New("b498572340593827", Bool(true))
	New("c094757438762023", Bool(false))

	logger := ktesting.NewLogger(t, ktesting.NewConfig(ktesting.BufferLogs(true)))

	// EXERCISE
	Log(logger)

	// VERIFY
	logEntries := getTestLoggerEntries(t, logger)
	g.Expect(logEntries).To(HaveLen(4))

	for _, logEntry := range logEntries {
		g.Expect(logEntry.Prefix).To(BeZero())
		g.Expect(logEntry.Type).To(Equal(ktesting.LogInfo))
		g.Expect(logEntry.Verbosity).To(Equal(0))
		g.Expect(logEntry.Message).To(Equal("Feature flag"))
		g.Expect(logEntry.Err).To(BeNil())
		g.Expect(logEntry.WithKVList).To(BeEmpty())
	}
	g.Expect(logEntries[0].ParameterKVList).To(HaveExactElements(
		"key", "a567943574457334",
		"enabled", false,
	))
	g.Expect(logEntries[1].ParameterKVList).To(HaveExactElements(
		"key", "b498572340593827",
		"enabled", true,
	))
	g.Expect(logEntries[2].ParameterKVList).To(HaveExactElements(
		"key", "c094757438762023",
		"enabled", false,
	))
	g.Expect(logEntries[3].ParameterKVList).To(HaveExactElements(
		"key", "d896063233040385",
		"enabled", true,
	))
}

func Test_Log_NoFlags(t *testing.T) {
	// SETUP
	g := NewGomegaWithT(t)

	origFlags := flags
	t.Cleanup(func() { flags = origFlags })
	flags = make(map[string]*FeatureFlag)

	logger := ktesting.NewLogger(t, ktesting.NewConfig(ktesting.BufferLogs(true)))

	// EXERCISE
	Log(logger)

	// VERIFY
	logEntries := getTestLoggerEntries(t, logger)
	g.Expect(logEntries).To(HaveLen(0))
}

func getTestLoggerEntries(t *testing.T, logger logr.Logger) ktesting.Log {
	t.Helper()

	underlyingLogger, ok := logger.GetSink().(ktesting.Underlier)
	if !ok {
		t.Fatalf("should have had ktesting LogSink, got %T", logger.GetSink())
	}
	return underlyingLogger.GetBuffer().Data()
}
