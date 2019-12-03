package k8s

import (
	"testing"

	"github.com/SAP/stewardci-core/pkg/k8s/fake"
	"gotest.tools/assert"
)

func Test_pipelineRunFetcher_ByName_NotExisting(t *testing.T) {
	// SETUP
	factory := fake.NewClientFactory()
	examinee := NewPipelineRunFetcher(factory)

	// EXERCISE
	pipelineRun, resultErr := examinee.ByName(ns1, "NotExisting1")

	// VERIFY
	assert.Assert(t, pipelineRun == nil)
	assert.NilError(t, resultErr)
}

func Test_pipelineRunFetcher_ByName_GoodCase(t *testing.T) {
	// SETUP
	const (
		secretName = "secret1"
	)
	run := newPipelineRunWithSecret(ns1, run1, secretName)
	factory := fake.NewClientFactory(run)
	examinee := NewPipelineRunFetcher(factory)

	// EXERCISE
	resultObj, resultErr := examinee.ByName(ns1, run1)

	// VERIFY
	assert.NilError(t, resultErr)
	assert.DeepEqual(t, run, resultObj)
}

func Test_pipelineRunFetcher_ByKey_GoodCase(t *testing.T) {
	// SETUP
	const (
		secretName = "secret1"
	)
	run := newPipelineRunWithSecret(ns1, run1, secretName)
	factory := fake.NewClientFactory(run)
	key := fake.ObjectKey(run1, ns1)
	examinee := NewPipelineRunFetcher(factory)

	// EXERCISE
	resultObj, resultErr := examinee.ByKey(key)

	// VERIFY
	assert.NilError(t, resultErr)
	assert.DeepEqual(t, run, resultObj)
}
