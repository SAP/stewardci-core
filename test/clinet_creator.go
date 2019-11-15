package test

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/SAP/stewardci-core/pkg/k8s"
	knativetest "knative.dev/pkg/test"
)

// Time to wait until the next resync takes place.
// Resync is only required if events got lost or if the controller restarted (and missed events).
const resyncPeriod = 5 * time.Minute

func setup(t *testing.T) (k8s.ClientFactory, string, Waiter) {
	t.Helper()
	kubeconfig := knativetest.Flags.Kubeconfig
	clusterName := knativetest.Flags.Cluster
	log.Printf("Create Factory (config: %s,resync period: %s)", kubeconfig, resyncPeriod.String())
	config, err := knativetest.BuildClientConfig(kubeconfig, clusterName)
	if err != nil {
		panic(err.Error())
	}
	factory := k8s.NewClientFactory(config, resyncPeriod)
	if factory == nil {
		t.Fatalf("failed to create client factory for config file '%s'.", kubeconfig)
	}
	testClient := os.Getenv("STEWARD_TEST_CLIENT")
	if testClient == "" {
		t.Fatalf("environment variable STEWARD_TEST_CLIENT undefined")
	}
	return factory, "steward-test-c", NewWaiter(factory)
}
