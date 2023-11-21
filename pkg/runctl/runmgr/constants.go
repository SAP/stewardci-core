package runmgr

import steward "github.com/SAP/stewardci-core/pkg/apis/steward"

const (
	// JFRTaskRunName is the name of the Tekton TaskRun in each
	// run namespace.
	JFRTaskRunName = "steward-jenkinsfile-runner"

	// JFRTaskRunStepName is the name of the step in the Tekton TaskRun that executes
	// the Jenkinsfile Runner
	JFRTaskRunStepName = "jenkinsfile-runner"
)

const (
	runNamespacePrefix       = "steward-run"
	runNamespaceRandomLength = 5
	serviceAccountName       = "default"
	serviceAccountTokenName  = "steward-serviceaccount-token"

	// in general, the token of the above service account should not be automatically mounted into pods
	automountServiceAccountToken = false

	annotationPipelineRunKey = steward.GroupName + "/pipeline-run-key"

	jfrResultKey string = "jfr-termination-log"
)
