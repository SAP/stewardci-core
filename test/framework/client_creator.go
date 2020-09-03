package framework

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/SAP/stewardci-core/pkg/k8s"
	klog "k8s.io/klog/v2"
	knativetest "knative.dev/pkg/test"
)

// Time to wait until the next resync takes place.
// Resync is only required if events got lost or if the controller restarted (and missed events).
const resyncPeriod = 5 * time.Minute

// Setup prepares the test environment
func Setup(t *testing.T) context.Context {
	t.Helper()
	kubeconfig := knativetest.Flags.Kubeconfig
	clusterName := knativetest.Flags.Cluster
	klog.V(3).Printf("Create Factory (config: %s,resync period: %s)", kubeconfig, resyncPeriod.String())
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
	tenantNamespace := os.Getenv("STEWARD_TEST_TENANT")
	ctx := context.Background()
	ctx = SetNamespace(ctx, testClient)
	ctx = SetTenantNamespace(ctx, tenantNamespace)
	ctx = SetClientFactory(ctx, factory)
	ctx = SetRealmUUID(ctx)
	klog.V(3).Printf("RealmUUID: %q", GetRealmUUID(ctx))
	return ctx
}
