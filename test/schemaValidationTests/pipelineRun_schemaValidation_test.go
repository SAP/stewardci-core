// +build e2e

package schemavalidationtests

import (
	"strings"
	"testing"

	framework "github.com/SAP/stewardci-core/test/framework"
	"gotest.tools/assert"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func Test_PipelineRunSchemaValidation(t *testing.T) {
	for _, test := range pipelineRunTests {
		t.Run(test.name, func(t *testing.T) {
			// PREPARE
			ctx := framework.Setup(t)

			// EXERCISE
			pipelineRun, err := framework.CreatePipelineRunFromYAML(ctx, test.data)
			defer framework.DeletePipelineRun(ctx, pipelineRun)

			// VERIFY
			test.check(t, err)
		})
	}
}

const pipelineRunHeaderYAML string = `
apiVersion: steward.sap.com/v1alpha1
kind: PipelineRun
metadata:
	generateName: test-pipelinerun-validation-
`

var pipelineRunTests = []SchemaValidationTest{

	// ###################################################################
	SchemaValidationTest{
		name:       "minimal good case",
		dataFormat: yaml,
		data: fixIndent(2, `%v
		spec:
			jenkinsFile:
				repoUrl: repoUrl1
				revision: revision1
				relativePath: relativePath1
			args: {}
			intent: run
			logging:
				elasticsearch:
					runID: {}
		`, pipelineRunHeaderYAML),
		check: func(t *testing.T, err error) {
			assert.NilError(t, err)
		},
	},

	// ###################################################################
	SchemaValidationTest{
		name:       "spec empty",
		dataFormat: yaml,
		data: fixIndent(2, `%v
		spec: {}
		`, pipelineRunHeaderYAML),
		check: func(t *testing.T, err error) {
			assert.ErrorContains(t, err, "spec.jenkinsFile in body is required")
			count := strings.Count(err.Error(), "spec.")
			assert.Assert(t, count == 1, "Unexpected number of validation failures: %v : %v ", count, err.Error())
		},
	},

	// ###################################################################
	SchemaValidationTest{
		name:       "spec missing",
		dataFormat: yaml,
		data:       fixIndent(0, `%v`, pipelineRunHeaderYAML),
		check: func(t *testing.T, err error) {
			assert.ErrorContains(t, err, ".spec in body is required")
		},
	},

	// ###################################################################
	SchemaValidationTest{
		name:       "spec.jenkinsFile entries missing",
		dataFormat: yaml,
		data: fixIndent(2, `%v
		spec:
			jenkinsFile: {}			#empty
			args: {}
			intent: run
			logging:
				elasticsearch:
					runID: {}
		`, pipelineRunHeaderYAML),
		check: func(t *testing.T, err error) {
			assert.ErrorContains(t, err, "spec.jenkinsFile.repoUrl in body is required")
			assert.ErrorContains(t, err, "spec.jenkinsFile.revision in body is required")
			assert.ErrorContains(t, err, "spec.jenkinsFile.relativePath in body is required")
			count := strings.Count(err.Error(), "spec.")
			assert.Assert(t, count == 3, "Unexpected number of validation failures: %v : %v ", count, err.Error())
		},
	},

	// ###################################################################
	SchemaValidationTest{
		name:       "spec entry values empty strings",
		dataFormat: yaml,
		data: fixIndent(2, `%v
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
		`, pipelineRunHeaderYAML),
		check: func(t *testing.T, err error) {
			assert.ErrorContains(t, err, "spec.jenkinsFile.repoUrl in body should match '^[^\\s]{1,}.*$'")
			assert.ErrorContains(t, err, "spec.jenkinsFile.revision in body should match '^[^\\s]{1,}.*$'")
			assert.ErrorContains(t, err, "spec.jenkinsFile.relativePath in body should match '^[^\\s]{1,}.*$'")
			assert.ErrorContains(t, err, "spec.args in body must be of type object: \"string\"")
			assert.ErrorContains(t, err, "spec.logging.elasticsearch.runID in body must be of type object: \"string\"")
			count := strings.Count(err.Error(), "spec.")
			assert.Assert(t, count == 5, "Unexpected number of validation failures: %v : %v ", count, err.Error())
		},
	},

	// ###################################################################
	SchemaValidationTest{
		name:       "spec entry values unset",
		dataFormat: yaml,
		data: fixIndent(2, `%v
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
		`, pipelineRunHeaderYAML),
		check: func(t *testing.T, err error) {
			assert.ErrorContains(t, err, "spec.jenkinsFile.relativePath in body must be of type string: \"null\"")
			assert.ErrorContains(t, err, "spec.jenkinsFile.repoUrl in body must be of type string: \"null\"")
			assert.ErrorContains(t, err, "spec.jenkinsFile.revision in body must be of type string: \"null\"")
			assert.ErrorContains(t, err, "spec.args in body must be of type object: \"null\"")
			assert.ErrorContains(t, err, "spec.intent in body must be of type string: \"null\"")
			assert.ErrorContains(t, err, "spec.logging.elasticsearch.runID in body must be of type object: \"null\"")
			count := strings.Count(err.Error(), "spec.")
			assert.Assert(t, count == 6, "Unexpected number of validation failures: %v : %v ", count, err.Error())
		},
	},

	// ###################################################################
	SchemaValidationTest{
		name:       "spec entry values invalid",
		dataFormat: yaml,
		data: fixIndent(2, `%v
		spec:
			jenkinsFile:
				repoUrl: repoUrl1
				revision: revision1
				relativePath: relativePath1
			args: {}
			intent: invalid
			logging:
				elasticsearch:
					runID: {}
		`, pipelineRunHeaderYAML),
		check: func(t *testing.T, err error) {
			assert.ErrorContains(t, err, "spec.intent in body should match '^(|run|abort)$'")
		},
	},
}
