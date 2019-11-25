# Integration tests

When executing tests make sure that --parallel option is set high enough.
Test writes a Parallel: x to the log indicating how many tests are runing in parallel.

## Prepare test client

```bash
export STEWARD_TEST_CLIENT=$(kubectl apply -f test-client.yaml -o=name)
export STEWARD_TEST_CLIENT=${STEWARD_TEST_CLIENT#*/}
```

## Run framework tests to check if the test framework works correctly

```bash
cd framework
go test -count=1 -tags=frameworktest -v --kubeconfig $KUBECONFIG .
```

## Run integration tests

```bash
go test -count=1 -tags=e2e -v --kubeconfig $KUBECONFIG .
```

## Run load tests

```bash
cd loadtest
go test -count=1 -tags=loadtest -v --kubeconfig $KUBECONFIG .
```

## Cleanup
```bash
kubectl delete $STEWARD_TEST_CLIENT
```
