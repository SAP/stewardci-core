package constants

import (
	"github.com/SAP/stewardci-core/pkg/k8s"
)

const (

	// RunClusterRoleName is the name of the cluster role
	RunClusterRoleName k8s.RoleName = "steward-run"

	// JFRStepName is the name of the jfs step
	JFRStepName = "step-jenkinsfile-runner"

	// TektonTaskRunName is the name of the Tekton TaskRun in each
	// run namespace.
	TektonTaskRunName = "steward-jenkinsfile-runner"
)
