package runctl

import (
	"github.com/SAP/stewardci-core/pkg/k8s"
)

const runClusterRoleName k8s.RoleName = "steward-run"

const defaultBuildTimeout = "60m"
