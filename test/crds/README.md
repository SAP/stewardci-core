# Tests of Steward Custom Resource Definitions

Test of Steward CRDs in a Kubernetes cluster.

## Prerequisites

Update the Steward CRDs in the test cluster by installing the Helm chart.

**The tests will use the _installed_ CRDs, not the ones in the source tree.**

## Running Tests

See [../README.md](../README.md).

From this folder test can be run with this command:

```sh
go test ./... -count=1 -tags=e2e -v -- --kubeconfig "$KUBECONFIG"
```
