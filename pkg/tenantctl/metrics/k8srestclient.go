package metrics

import (
	// activate embedding of client-go rest client metrics
	_ "github.com/SAP/stewardci-core/pkg/metrics/k8srestclient"
)
