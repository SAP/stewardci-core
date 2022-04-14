//go:build e2e
// +build e2e

package pipelinerun

import (
	"context"
	"testing"

	stewardv1alpha1 "github.com/SAP/stewardci-core/pkg/apis/steward/v1alpha1"
	framework "github.com/SAP/stewardci-core/test/framework"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const pipelineRunHeader string = `
apiVersion: steward.sap.com/v1alpha1
kind: PipelineRun
metadata:
	generateName: test-pipelinerun-validation-
	labels:
		steward.sap.com/ignore: ""
`

const minimalPipelineRunSpec string = `
# keep indentation
	jenkinsFile:
		repoUrl: repoUrl1
		revision: revision1
		relativePath: relativePath1
`

func Test_CRDs_PipelineRun_Schema_Spec(t *testing.T) {

	testcases := []struct {
		name  string
		spec  string
		check func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error)
	}{
		//
		// Do not match against error messages, as they change between
		// Kubernetes versions.
		//
		// Instead, failure tests should vary only a single field compared
		// to a success test and only check for the varied key name in the
		// error message.
		//

		/////////////////////////////////////////////////////////////////
		// minimum
		/////////////////////////////////////////////////////////////////

		{
			name: "minimal good case",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec
		/////////////////////////////////////////////////////////////////

		{
			name: "spec missing",
			spec: "# no spec here",
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec"))
			},
		},

		{
			name: "spec empty",
			spec: fixIndent(`
				spec: {}
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.jenkinsFile.repoUrl
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.jenkinsFile.repoUrl missing",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						#repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.repoUrl"))
			},
		},

		{
			name: "spec.jenkinsFile.repoUrl null",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: null  # not allowed
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.repoUrl"))
			},
		},

		{
			name: "spec.jenkinsFile.repoUrl empty",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: ""  # not allowed
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.repoUrl"))
			},
		},

		{
			name: "spec.jenkinsFile.repoUrl invalid value",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: " abc"  # not allowed
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.repoUrl"))
			},
		},

		{
			name: "spec.jenkinsFile.repoUrl invalid type",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: 1  # not allowed
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.repoUrl"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.jenkinsFile.revision
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.jenkinsFile.revision missing",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: repoUrl1
						#revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.revision"))
			},
		},

		{
			name: "spec.jenkinsFile.revision null",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: repoUrl1
						revision: null  # not allowed
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.revision"))
			},
		},

		{
			name: "spec.jenkinsFile.revision empty",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: repoUrl1
						revision: ""  # not allowed
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.revision"))
			},
		},

		{
			name: "spec.jenkinsFile.revision invalid value",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: repoUrl1
						revision: " abc"  # not allowed
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.revision"))
			},
		},

		{
			name: "spec.jenkinsFile.revision invalid type",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: repoUrl1
						revision: 1  # not allowed
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.revision"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.jenkinsFile.relativePath
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.jenkinsFile.relativePath missing",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						#relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.relativePath"))
			},
		},

		{
			name: "spec.jenkinsFile.relativePath null",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: null  # not allowed
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.relativePath"))
			},
		},

		{
			name: "spec.jenkinsFile.relativePath empty",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: ""  # not allowed
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.relativePath"))
			},
		},

		{
			name: "spec.jenkinsFile.relativePath invalid value",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: " abc"  # not allowed
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.relativePath"))
			},
		},

		{
			name: "spec.jenkinsFile.relativePath invalid type",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: 1  # not allowed
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.relativePath"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.jenkinsFile.repoAuthSecret
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.jenkinsFile.repoAuthSecret null",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoAuthSecret: null
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.jenkinsFile.repoAuthSecret empty",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoAuthSecret: ""
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.jenkinsFile.repoAuthSecret invalid type",
			spec: fixIndent(`
				spec:
					jenkinsFile:
						repoAuthSecret: 1  # not allowed
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsFile.repoAuthSecret"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.jenkinsfileRunner
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.jenkinsfileRunner empty",
			spec: fixIndent(`
				spec:
					jenkinsfileRunner: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.jenkinsfileRunner invalid type",
			spec: fixIndent(`
				spec:
					jenkinsfileRunner: []  # invalid type
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsfileRunner"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.jenkinsfileRunner.image
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.jenkinsfileRunner.image null",
			spec: fixIndent(`
				spec:
					jenkinsfileRunner:
						image: null
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.jenkinsfileRunner.image empty",
			spec: fixIndent(`
				spec:
					jenkinsfileRunner:
						image: ""  # not allowed
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.jenkinsfileRunner.image invalid type",
			spec: fixIndent(`
				spec:
					jenkinsfileRunner:
						image: 1  # not allowed
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsfileRunner.image"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.jenkinsfileRunner.imagePullPolicy
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.jenkinsfileRunner.imagePullPolicy null",
			spec: fixIndent(`
				spec:
					jenkinsfileRunner:
						imagePullPolicy: null
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.jenkinsfileRunner.imagePullPolicy empty",
			spec: fixIndent(`
				spec:
					jenkinsfileRunner:
						imagePullPolicy: ""
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.jenkinsfileRunner.imagePullPolicy value Never",
			spec: fixIndent(`
				spec:
					jenkinsfileRunner:
						imagePullPolicy: Never
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.jenkinsfileRunner.imagePullPolicy value IfNotPresent",
			spec: fixIndent(`
				spec:
					jenkinsfileRunner:
						imagePullPolicy: IfNotPresent
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.jenkinsfileRunner.imagePullPolicy value Always",
			spec: fixIndent(`
				spec:
					jenkinsfileRunner:
						imagePullPolicy: Always
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.jenkinsfileRunner.imagePullPolicy invalid value",
			spec: fixIndent(`
				spec:
					jenkinsfileRunner:
						imagePullPolicy: "never"  # not allowed
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsfileRunner.imagePullPolicy"))
			},
		},

		{
			name: "spec.jenkinsfileRunner.imagePullPolicy invalid type",
			spec: fixIndent(`
				spec:
					jenkinsfileRunner:
						imagePullPolicy: 1  # not allowed
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.jenkinsfileRunner.imagePullPolicy"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.args
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.args empty",
			spec: fixIndent(`
				spec:
					args: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.args invalid type",
			spec: fixIndent(`
				spec:
					args: []
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.args"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.args.*
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.args.* good case",
			spec: fixIndent(`
				spec:
					args:
						key1: value1
						key2: value2
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.args.* empty",
			spec: fixIndent(`
				spec:
					args:
						key1: value1
						"": value2  # empty key
						key3: ""    # empty value
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.args.* invalid type",
			spec: fixIndent(`
				spec:
					args:
						key1: value1
						key2: 1  # invalid type
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, errorContainsToken(resultErr, "spec.args.key2"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.secrets
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.secrets empty",
			spec: fixIndent(`
				spec:
					secrets: []
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.secrets invalid type",
			spec: fixIndent(`
				spec:
					secrets: {}  # invalid type
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.secrets"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.secrets.*
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.secrets.* filled",
			spec: fixIndent(`
				spec:
					secrets:
						- secret1
						- secret2
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.secrets.* null",
			spec: fixIndent(`
				spec:
					secrets:
						- null
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.secrets"))
			},
		},

		{
			name: "spec.secrets.* empty",
			spec: fixIndent(`
				spec:
					secrets:
						- ""
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.secrets"))
			},
		},

		{
			name: "spec.secrets.* invalid value",
			spec: fixIndent(`
				spec:
					secrets:
						- " abc"  # not allowed
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.secrets"))
			},
		},

		{
			name: "spec.secrets.* invalid type",
			spec: fixIndent(`
				spec:
					secrets:
						- 1  # not allowed
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.secrets"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.imagePullSecrets
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.imagePullSecrets empty",
			spec: fixIndent(`
				spec:
					imagePullSecrets: []
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.imagePullSecrets invalid type",
			spec: fixIndent(`
				spec:
					imagePullSecrets: {}  # invalid type
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.imagePullSecrets"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.imagePullSecrets.*
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.imagePullSecrets.* filled",
			spec: fixIndent(`
				spec:
					imagePullSecrets:
						- secret1
						- secret2
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.imagePullSecrets.* null",
			spec: fixIndent(`
				spec:
					imagePullSecrets:
						- null
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.imagePullSecrets"))
			},
		},

		{
			name: "spec.imagePullSecrets.* empty",
			spec: fixIndent(`
				spec:
					imagePullSecrets:
						- ""
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.imagePullSecrets"))
			},
		},

		{
			name: "spec.imagePullSecrets.* invalid value",
			spec: fixIndent(`
				spec:
					imagePullSecrets:
						- " abc"  # not allowed
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.imagePullSecrets"))
			},
		},

		{
			name: "spec.imagePullSecrets.* invalid type",
			spec: fixIndent(`
				spec:
					imagePullSecrets:
						- 1  # not allowed
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.imagePullSecrets"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.intent
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.intent null",
			spec: fixIndent(`
				spec:
					intent: null
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.intent empty",
			spec: fixIndent(`
				spec:
					intent: ""
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.intent value run",
			spec: fixIndent(`
				spec:
					intent: run
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.intent value abort",
			spec: fixIndent(`
				spec:
					intent: abort
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.intent invalid value",
			spec: fixIndent(`
				spec:
					intent: RUN  # invalid value
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.intent"))
			},
		},

		{
			name: "spec.intent invalid type",
			spec: fixIndent(`
				spec:
					intent: 1  # invalid type
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.intent"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.logging
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.logging null",
			spec: fixIndent(`
				spec:
					logging: null
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.logging empty",
			spec: fixIndent(`
				spec:
					logging: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.logging invalid type",
			spec: fixIndent(`
				spec:
					logging: []  # invalid type
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.logging"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.logging.elasticsearch
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.logging.elasticsearch null",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch: null
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.logging.elasticsearch empty",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch: {}  # missing subkey
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.logging.elasticsearch."))
			},
		},

		{
			name: "spec.logging.elasticsearch invalid type",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch: []  # invalid type
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.logging.elasticsearch"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.logging.elasticsearch.runID
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.logging.elasticsearch.runID null",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch:
							runID: null  # not allowed
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.logging.elasticsearch.runID"))
			},
		},

		{
			name: "spec.logging.elasticsearch.runID empty",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch:
							runID: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.logging.elasticsearch.runID valid value",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch:
							runID:
								null1: null
								bool1: true
								string1: ""
								string2: "x"
								number1: 1
								number2: 1.5
								array1:
								- null
								- true
								- ""
								- x
								- 1
								- 1.5
								- []
								- {}
								object1:
									null1: null
									bool1: true
									string1: ""
									string2: "x"
									number1: 1
									number2: 1.5
									array1: []
									object1: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)

				expectedStatus := map[string]interface{}{
					"null1":   nil,
					"bool1":   true,
					"string1": "",
					"string2": "x",
					"number1": 1.0,
					"number2": 1.5,
					"array1":  []interface{}{nil, true, "", "x", 1.0, 1.5, []interface{}{}, map[string]interface{}{}},
					"object1": map[string]interface{}{
						"null1":   nil,
						"bool1":   true,
						"string1": "",
						"string2": "x",
						"number1": 1.0,
						"number2": 1.5,
						"array1":  []interface{}{},
						"object1": map[string]interface{}{},
					},
				}
				assert.DeepEqual(t, expectedStatus, result.Spec.Logging.Elasticsearch.RunID.Value)
			},
		},

		{
			name: "spec.logging.elasticsearch.runID invalid type",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch:
							runID: 1  # not allowed
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.logging.elasticsearch.runID"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.logging.elasticsearch.indexURL
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.logging.elasticsearch.indexURL null",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch:
							indexURL: null
							runID: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.logging.elasticsearch.indexURL empty",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch:
							indexURL: ""
							runID: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.logging.elasticsearch.indexURL valid value",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch:
							indexURL: "abc"
							runID: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.logging.elasticsearch.indexURL invalid type",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch:
							indexURL: 1  # invalid type
							runID: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.logging.elasticsearch.indexURL"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.logging.elasticsearch.authSecret
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.logging.elasticsearch.authSecret null",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch:
							authSecret: null
							runID: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.logging.elasticsearch.authSecret empty",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch:
							authSecret: ""
							runID: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.logging.elasticsearch.authSecret valid value",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch:
							authSecret: "abc"
							runID: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.logging.elasticsearch.authSecret invalid type",
			spec: fixIndent(`
				spec:
					logging:
						elasticsearch:
							authSecret: 1  # invalid type
							runID: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.logging.elasticsearch.authSecret"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.runDetails
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.runDetails null",
			spec: fixIndent(`
				spec:
					runDetails: null
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.runDetails empty",
			spec: fixIndent(`
				spec:
					runDetails: {}
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.runDetails invalid type",
			spec: fixIndent(`
				spec:
					runDetails: []  # invalid type
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.runDetails"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.runDetails.jobName
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.runDetails.jobName null",
			spec: fixIndent(`
				spec:
					runDetails:
						jobName: null
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.runDetails.jobName empty",
			spec: fixIndent(`
				spec:
					runDetails:
						jobName: ""
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.runDetails.jobName valid value",
			spec: fixIndent(`
				spec:
					runDetails:
						jobName: "abc"
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.runDetails.jobName invalid type",
			spec: fixIndent(`
				spec:
					runDetails:
						jobName: 1  # invalid type
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.runDetails.jobName"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.runDetails.sequenceNumber
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.runDetails.sequenceNumber min valid value",
			spec: fixIndent(`
				spec:
					runDetails:
						sequenceNumber: 0
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.runDetails.sequenceNumber valid value 1",
			spec: fixIndent(`
				spec:
					runDetails:
						sequenceNumber: 1
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.runDetails.sequenceNumber max valid value",
			spec: fixIndent(`
				spec:
					runDetails:
						sequenceNumber: 2147483647
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.runDetails.sequenceNumber invalid value less than min",
			spec: fixIndent(`
				spec:
					runDetails:
						sequenceNumber: -1  # invalid value
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.runDetails.sequenceNumber"))
			},
		},

		{
			name: "spec.runDetails.sequenceNumber invalid value greater than max",
			spec: fixIndent(`
				spec:
					runDetails:
						sequenceNumber: 2147483648
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.runDetails.sequenceNumber"))
			},
		},

		{
			name: "spec.runDetails.sequenceNumber null",
			spec: fixIndent(`
				spec:
					runDetails:
						sequenceNumber: null
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.runDetails.sequenceNumber invalid type string",
			spec: fixIndent(`
				spec:
					runDetails:
						sequenceNumber: "1"  # invalid type
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.runDetails.sequenceNumber"))
			},
		},

		{
			name: "spec.runDetails.sequenceNumber invalid type float",
			spec: fixIndent(`
				spec:
					runDetails:
						sequenceNumber: 1.5  # invalid type
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.runDetails.sequenceNumber"))
			},
		},

		/////////////////////////////////////////////////////////////////
		// spec.runDetails.cause
		/////////////////////////////////////////////////////////////////

		{
			name: "spec.runDetails.cause null",
			spec: fixIndent(`
				spec:
					runDetails:
						cause: null
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.runDetails.cause empty",
			spec: fixIndent(`
				spec:
					runDetails:
						cause: ""
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.runDetails.cause valid value",
			spec: fixIndent(`
				spec:
					runDetails:
						cause: "abc"
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.NilError(t, resultErr)
			},
		},

		{
			name: "spec.runDetails.cause invalid type",
			spec: fixIndent(`
				spec:
					runDetails:
						cause: 1  # invalid type
					jenkinsFile:
						repoUrl: repoUrl1
						revision: revision1
						relativePath: relativePath1
			`),
			check: func(t *testing.T, result *stewardv1alpha1.PipelineRun, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "spec.runDetails.cause"))
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			testcase := testcase
			t.Parallel()

			// SETUP
			ctx := framework.Setup(t)

			manifest := fixIndent(pipelineRunHeader + testcase.spec)

			// EXERCISE
			result, resultErr := framework.CreatePipelineRunFromYAML(ctx, manifest)

			if result != nil {
				defer framework.DeletePipelineRun(ctx, result)
			}

			// VERIFY
			testcase.check(t, result, resultErr)
		})
	}
}

func Test_CRDs_PipelineRun_Schema_Status(t *testing.T) {

	testcases := []struct {
		name   string
		status string
		check  func(t *testing.T, result *unstructured.Unstructured, resultErr error)
	}{
		//
		// Do not match against error messages, as they change between
		// Kubernetes versions.
		//
		// Instead, failure tests should vary only a single field compared
		// to a success test and only check for the varied key name in the
		// error message.
		//

		/////////////////////////////////////////////////////////////////
		// status
		/////////////////////////////////////////////////////////////////

		{
			name: "status null",
			status: fixIndent(`
				status: null
			`),
			check: func(t *testing.T, result *unstructured.Unstructured, resultErr error) {
				assert.NilError(t, resultErr)
				if _, ok := result.UnstructuredContent()["status"]; ok {
					t.Fatal("field \"status\" exists but should not")
				}
			},
		},

		{
			name: "status empty",
			status: fixIndent(`
				status: {}
			`),
			check: func(t *testing.T, result *unstructured.Unstructured, resultErr error) {
				assert.NilError(t, resultErr)
				status, exists := result.UnstructuredContent()["status"]
				if !exists {
					t.Fatal("status field does not exist")
				}
				assert.DeepEqual(t, map[string]interface{}{}, status)
			},
		},

		{
			name: "status valid value",
			status: fixIndent(`
				status:
					null1: null
					bool1: true
					string1: ""
					string2: "x"
					number1: 1
					number2: 1.5
					array1:
					- null
					- true
					- ""
					- x
					- 1
					- 1.5
					- []
					- {}
					object1:
						null1: null
						bool1: true
						string1: ""
						string2: "x"
						number1: 1
						number2: 1.5
						array1: []
						object1: {}
			`),
			check: func(t *testing.T, result *unstructured.Unstructured, resultErr error) {
				assert.NilError(t, resultErr)

				status := result.UnstructuredContent()["status"]
				expectedStatus := map[string]interface{}{
					"null1":   nil,
					"bool1":   true,
					"string1": "",
					"string2": "x",
					"number1": int64(1),
					"number2": 1.5,
					"array1":  []interface{}{nil, true, "", "x", int64(1), 1.5, []interface{}{}, map[string]interface{}{}},
					"object1": map[string]interface{}{
						"null1":   nil,
						"bool1":   true,
						"string1": "",
						"string2": "x",
						"number1": int64(1),
						"number2": 1.5,
						"array1":  []interface{}{},
						"object1": map[string]interface{}{},
					},
				}
				assert.DeepEqual(t, expectedStatus, status)
			},
		},

		{
			name: "status invalid type",
			status: fixIndent(`
				status: 1  # not allowed
			`),
			check: func(t *testing.T, result *unstructured.Unstructured, resultErr error) {
				assert.Assert(t, resultErr != nil)
				assert.Assert(t, errorContainsToken(resultErr, "status"))
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			testcase := testcase
			t.Parallel()

			// SETUP
			ctx := framework.Setup(t)

			// writing the status subresource requires an existing main resource
			initialObjectYAML := fixIndent(pipelineRunHeader + "spec:\n" + minimalPipelineRunSpec)
			initialObject, err := createPipelineRunFromYAML(ctx, initialObjectYAML)
			assert.NilError(t, err)
			defer deleteResourceObject(ctx, "pipelineruns", initialObject.GetNamespace(), initialObject.GetName())

			var baseManifestYAML string
			{
				obj := initialObject.DeepCopy()
				objContent := obj.UnstructuredContent()

				delete(objContent, "status")
				objContent["apiVersion"] = stewardv1alpha1.SchemeGroupVersion.String()
				objContent["kind"] = "PipelineRun"

				manifestYAMLBytes, err := yaml.Marshal(objContent)
				assert.NilError(t, err)
				baseManifestYAML = string(manifestYAMLBytes)
			}

			manifestYAML := fixIndent(baseManifestYAML + testcase.status)

			// EXERCISE
			result, resultErr := updatePipelineRunStatusFromYAML(ctx, initialObject.GetNamespace(), initialObject.GetName(), manifestYAML)

			// VERIFY
			testcase.check(t, result, resultErr)
		})
	}
}

// CreatePipelineRunStatusFromYAML creates a pipeline run object via the status subresource
// using the given pipeline run manifest in YAML format.
func updatePipelineRunStatusFromYAML(ctx context.Context, namespace, name, pipelineRunYAML string) (result *unstructured.Unstructured, err error) {
	client := framework.GetClientFactory(ctx).StewardV1alpha1().RESTClient()
	result = &unstructured.Unstructured{}
	err = client.Put().
		Namespace(namespace).
		Resource("pipelineruns").
		SubResource("status").
		Name(name).
		Body([]byte(pipelineRunYAML)).
		SetHeader("Content-Type", "application/yaml").
		Do(ctx).
		Into(result)
	if err != nil {
		result = nil
	}
	return
}

func createPipelineRunFromYAML(ctx context.Context, pipelineRunYAML string) (result *unstructured.Unstructured, err error) {
	client := framework.GetClientFactory(ctx).StewardV1alpha1().RESTClient()
	result = &unstructured.Unstructured{}
	err = client.Post().
		Namespace(framework.GetNamespace(ctx)).
		Resource("pipelineruns").
		Body([]byte(pipelineRunYAML)).
		SetHeader("Content-Type", "application/yaml").
		Do(ctx).
		Into(result)
	if err != nil {
		result = nil
	}
	return
}

func deleteResourceObject(ctx context.Context, resource, namespace, name string) error {
	dynamicIfce := framework.GetClientFactory(ctx).
		Dynamic().
		Resource(stewardv1alpha1.SchemeGroupVersion.WithResource("pipelineruns")).
		Namespace(namespace)
	return dynamicIfce.Delete(ctx, name, v1.DeleteOptions{})
}
