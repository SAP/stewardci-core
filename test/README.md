# Test

When executing tests make sure that --parallel option is set high enough.
Test writes a Parallel: x to the log indicating how many tests are runing in parallel.

## Prepare test client

```bash
export STEWARD_TEST_CLIENT=$(kubectl apply -f test-client.yaml -o=name)
export STEWARD_TEST_CLIENT=${STEWARD_TEST_CLIENT#*/}
```

## Run integration tests

```bash
export STEWARD_TEST_CLIENT=$(kubectl apply -f test-client.yaml -o=name)

go test -count=1 -tags=e2e --parallel 10 -v --kubeconfig $KUBECONFIG .
```

## Run load tests

```bash
go test -count=1 -tags=loadtest --parallel 10 -v --kubeconfig $KUBECONFIG .
```

## Cleanup
```bash
kubectl delete $STEWARD_TEST_CLIENT
```
