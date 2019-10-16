package runctl

import (
	"github.com/SAP/stewardci-core/pkg/k8s"
)

//const pullSecretName string = "docker-pull"
//const scmCloneSecretName string = "github-tools"

// Don't use predefined secret names
const pullSecretName string = ""
const scmCloneSecretName string = ""

const runClusterRoleName k8s.RoleName = "steward-run"

const defaultBuildTimeout = "60m"
