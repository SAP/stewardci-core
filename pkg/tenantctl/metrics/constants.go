package metrics

import "github.com/SAP/stewardci-core/pkg/metrics"

const (
	subsystem = metrics.Subsystem + "_tenants"
)

var _ = subsystem // avoid unused warning
