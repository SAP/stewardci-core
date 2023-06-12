# Integration Tests

Tests that need a Kubernetes cluster with Steward installed.

## Preparation

### Prepare Test Namespace

```bash
STEWARD_TEST_NAMESPACE=$(kubectl create -f - -o name <<<'{ "apiVersion": "v1", "kind": "Namespace", "metadata": { "generateName": "steward-test-" } }')
export STEWARD_TEST_NAMESPACE=${STEWARD_TEST_NAMESPACE#*/}
```

### Running Framework Tests

Framework tests test the test framework itself.

Running the test framework tests:

```bash
( cd framework && \
  kubectl -n "$STEWARD_TEST_NAMESPACE" delete secrets --all --ignore-not-found && \
  go test ./... -count=1 -tags=frameworktest -v -- --kubeconfig "$KUBECONFIG" )
```

## Running Tests

### Integration Tests

```bash
( cd integrationtest && \
  kubectl -n "$STEWARD_TEST_NAMESPACE" delete secrets --all --ignore-not-found && \
  go test ./... -count=1 -tags=e2e -run Test_PipelineRunSingle -v -- --kubeconfig "$KUBECONFIG" )
```

```bash
( cd crds && \
  kubectl -n "$STEWARD_TEST_NAMESPACE" delete secrets --all --ignore-not-found && \
  go test ./... -count=1 -tags=e2e -v -- --kubeconfig "$KUBECONFIG" )
```

### Load Tests

```bash
( cd loadtest && \
  kubectl -n "$STEWARD_TEST_NAMESPACE" delete secrets --all --ignore-not-found && \
  go test ./... -count=1 -tags=loadtest -v -- --kubeconfig "$KUBECONFIG" )
```

## Cleanup

```bash
kubectl delete namespace "$STEWARD_TEST_NAMESPACE"
```
