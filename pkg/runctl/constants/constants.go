package constants

import (
	"github.com/SAP/stewardci-core/pkg/k8s"
)

const (
	// RunClusterRoleName is the name of the cluster role that
	// pipeline run service accounts are bound to.
	RunClusterRoleName k8s.RoleName = "steward-run"
)
