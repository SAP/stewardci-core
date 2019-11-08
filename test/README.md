# Test

## Run integration tests

```bash
export STEWARD_TEST_CLIENT=$(kubectl apply -f test-client.yaml -o=name)

go test -count=1 -tags=e2e --kubeconfig $KUBECONFIG .
```

## Cleanup
```bash
kubectl delete $STEWARD_TEST_CLIENT
```
