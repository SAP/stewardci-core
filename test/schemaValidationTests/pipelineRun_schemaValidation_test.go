// +build e2e

package schemavalidationtests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	framework "github.com/SAP/stewardci-core/test/framework"
	"github.com/lithammer/dedent"
	"gotest.tools/assert"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func Test_PipelineRunSchemaValidation(t *testing.T) {
	data, checks := getPipelineRunSchemaTestData()

	for testName, testYAML := range data {
		t.Run(testName, func(t *testing.T) {
			// PREPARE
			ctx := framework.Setup(t)

			// EXERCISE
			pipelineRun, err := framework.CreatePipelineRunFromYAML(ctx, testYAML)
			defer framework.DeletePipelineRun(ctx, pipelineRun)

			// VERIFY
			check := checks[testName]
			check.(func(t *testing.T, pipelineRun *v1alpha1.PipelineRun, err error))(t, pipelineRun, err)
		})
	}
}

const pipelineRunHeaderYAML string = `
apiVersion: steward.sap.com/v1alpha1
kind: PipelineRun
metadata:
	generateName: test-pipelinerun-validation-
`

func getPipelineRunSchemaTestData() (data map[string]string, checks map[string]interface{}) {
	data = map[string]string{}
	checks = map[string]interface{}{}
	var testName string

	// good case #################
	testName = "minimal good case"
	data[testName] = fixIndent(fmt.Sprintf(`%v
spec:
	jenkinsFile:
		repoUrl: repoUrl1
		revision: revision1
		relativePath: relativePath1
	args: {}
	intent: intent1
	logging:
		elasticsearch:
			runID: {}
	`, pipelineRunHeaderYAML))
	checks[testName] = func(t *testing.T, pipelineRun *v1alpha1.PipelineRun, err error) {
		assert.NilError(t, err)
	}

	// spec #################
	testName = "spec empty"
	data[testName] = fixIndent(fmt.Sprintf(`%v
spec: {}
	`, pipelineRunHeaderYAML))
	checks[testName] = func(t *testing.T, pipelineRun *v1alpha1.PipelineRun, err error) {
		assert.ErrorContains(t, err, "spec.jenkinsFile in body is required")
		assert.ErrorContains(t, err, "spec.args in body is required")
		assert.ErrorContains(t, err, "spec.intent in body is required")
		assert.ErrorContains(t, err, "spec.logging in body is required")
		count := strings.Count(err.Error(), "spec.")
		assert.Assert(t, count == 4, "Unexpected number of validation failures: %v : %v ", count, err.Error())
	}

	testName = "spec missing"
	data[testName] = fixIndent(fmt.Sprintf(`%v`, pipelineRunHeaderYAML))
	checks[testName] = func(t *testing.T, pipelineRun *v1alpha1.PipelineRun, err error) {
		assert.ErrorContains(t, err, ".spec in body is required")
	}

	testName = "spec.jenkinsFile entries missing"
	data[testName] = fixIndent(fmt.Sprintf(`%v
spec:
	jenkinsFile: {}			#empty
	args: {}
	intent: intent1
	logging:
		elasticsearch:
			runID: {}
	`, pipelineRunHeaderYAML))
	checks[testName] = func(t *testing.T, pipelineRun *v1alpha1.PipelineRun, err error) {
		assert.ErrorContains(t, err, "spec.jenkinsFile.repoUrl in body is required")
		assert.ErrorContains(t, err, "spec.jenkinsFile.revision in body is required")
		assert.ErrorContains(t, err, "spec.jenkinsFile.relativePath in body is required")
		count := strings.Count(err.Error(), "spec.")
		assert.Assert(t, count == 3, "Unexpected number of validation failures: %v : %v ", count, err.Error())
	}

	testName = "spec entry keys empty strings"
	data[testName] = fixIndent(fmt.Sprintf(`%v
spec:
	jenkinsFile:
		repoUrl: "" 		#empty
		revision: "" 		#empty
		relativePath: "" 	#empty
	args: "" 				#empty
	intent: "" 				#empty
	logging:
		elasticsearch:
			runID: "" 		#empty
	`, pipelineRunHeaderYAML))
	checks[testName] = func(t *testing.T, pipelineRun *v1alpha1.PipelineRun, err error) {
		assert.ErrorContains(t, err, "spec.jenkinsFile.repoUrl in body should match '^[^\\s]{1,}.*$'")
		assert.ErrorContains(t, err, "spec.jenkinsFile.revision in body should match '^[^\\s]{1,}.*$'")
		assert.ErrorContains(t, err, "spec.jenkinsFile.relativePath in body should match '^[^\\s]{1,}.*$'")
		assert.ErrorContains(t, err, "spec.args in body must be of type object: \"string\"")
		assert.ErrorContains(t, err, "spec.intent in body should match '^[^\\s]{1,}.*$'")
		assert.ErrorContains(t, err, "spec.logging.elasticsearch.runID in body must be of type object: \"string\"")
		count := strings.Count(err.Error(), "spec.")
		assert.Assert(t, count == 6, "Unexpected number of validation failures: %v : %v ", count, err.Error())
	}

	testName = "spec entry values unset"
	data[testName] = fixIndent(fmt.Sprintf(`%v
spec:
	jenkinsFile:
		repoUrl:			#unset 
		revision: 			#unset
		relativePath: 		#unset
	args: 					#unset
	intent:					#unset
	logging:
		elasticsearch:
			runID: 			#unset
	`, pipelineRunHeaderYAML))
	checks[testName] = func(t *testing.T, pipelineRun *v1alpha1.PipelineRun, err error) {
		assert.ErrorContains(t, err, "spec.jenkinsFile.relativePath in body must be of type string: \"null\"")
		assert.ErrorContains(t, err, "spec.jenkinsFile.repoUrl in body must be of type string: \"null\"")
		assert.ErrorContains(t, err, "spec.jenkinsFile.revision in body must be of type string: \"null\"")
		assert.ErrorContains(t, err, "spec.args in body must be of type object: \"null\"")
		assert.ErrorContains(t, err, "spec.intent in body must be of type string: \"null\"")
		assert.ErrorContains(t, err, "spec.logging.elasticsearch.runID in body must be of type object: \"null\"")
		count := strings.Count(err.Error(), "spec.")
		assert.Assert(t, count == 6, "Unexpected number of validation failures: %v : %v ", count, err.Error())
	}

	return
}

// fixIndent removes common leading whitespace from all lines
// and replaces all tabs by spaces
func fixIndent(s string) (out string) {
	const TAB = "   "
	out = s
	out = dedent.Dedent(out)
	out = strings.ReplaceAll(out, "\t", TAB)
	return
}
