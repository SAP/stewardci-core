// +build e2e

package test

import (
	"log"
	"testing"
	"time"

	"github.com/SAP/stewardci-core/pkg/k8s"
	knativetest "knative.dev/pkg/test"
)

// Time to wait until the next resync takes place.
// Resync is only required if events got lost or if the controller restarted (and missed events).
const resyncPeriod = 5 * time.Minute

func setup(t *testing.T) (k8s.ClientFactory, string) {
	t.Helper()
	kubeconfig := knativetest.Flags.Kubeconfig
	log.Printf("Create Factory (resync period: %s)", resyncPeriod.String())
	factory := k8s.NewClientFactory(kubeconfig, resyncPeriod)
	if factory == nil {
		t.Fatalf("failed to create client factory for config file '%s'.", kubeconfig)
	}

	nsm := k8s.NewNamespaceManager(factory, "test", 6)
	namespace, err := nsm.Create("client", map[string]string{})
	if err != nil {
		t.Fatal("failed to create namespace for tests")
	}
	return factory, namespace
}
