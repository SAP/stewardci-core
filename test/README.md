# Integration Tests

Tests that need a Kubernetes cluster with Steward installed.

## Preparation

### Prepare Test Namespace

```bash
export STEWARD_TEST_NAMESPACE=steward-test
kubectl create ns "$STEWARD_TEST_NAMESPACE"
```

### Running Framework Tests

Framework tests test the test framework itself.

Running the test framework tests:

```bash
kubectl -n "$STEWARD_TEST_NAMESPACE" delete secret --all
( cd framework && go test ./... -count=1 -tags=frameworktest -v -- --kubeconfig "$KUBECONFIG" )
```

## Running Tests

### Integration Tests

Integration tests are split into tree groups to avoid client-side throttling and too much load on the test cluster.

```bash
kubectl -n "$STEWARD_TEST_NAMESPACE" delete secret --all
( cd integrationtest && \
  go test ./... -count=1 -tags=e2e -run Test_PipelineRunSingle -v -- --kubeconfig "$KUBECONFIG" )
```

```bash
kubectl -n "$STEWARD_TEST_NAMESPACE" delete secret --all
( cd crds && go test ./... -count=1 -tags=e2e -v -- --kubeconfig "$KUBECONFIG" )
```

### Load Tests

```bash
kubectl -n "$STEWARD_TEST_NAMESPACE" delete secret --all
( cd loadtest && go test ./... -count=1 -tags=loadtest -v -- --kubeconfig "$KUBECONFIG" )
```

## Cleanup

```bash
kubectl delete namespace "$STEWARD_TEST_NAMESPACE"
```
