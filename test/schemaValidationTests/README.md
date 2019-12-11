# Schema Validation Tests

The tests here are integration tests testing the schema validation of (the installed) Steward CRDs in an existing cluster.

## Prerequisited

Update the [PipelineRun](../../backend-k8s/steward-system/101-customResourceDefinition_PipelineRun.yaml) and [Tenant](../../backend-k8s/steward-system/101-customResourceDefinition_Tenant.yaml) Custom Resource Definitions in the test cluster. The tests will use the schema validation of the installed CRDs, not the ones in the sources here.

```sh
kubectl apply -f ../../backend-k8s/steward-system
```

## Test

See [README.md](../README.md) in parent folder.

```sh
go test -count=1 -tags=e2e -v --kubeconfig $KUBECONFIG .
```
