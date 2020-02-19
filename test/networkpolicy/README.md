# Integration tests for network policies

You need to be in the directory of this README to execute this tests.

## Preparation

```bash
# create client
export STEWARD_TEST_CLIENT=$(kubectl apply -f test-client.yaml -o=name)
export STEWARD_TEST_CLIENT=${STEWARD_TEST_CLIENT#*/}
```

## Run tests

```bash
./run_test.sh
```

## Cleanup

```bash
# delete client
kubectl delete namespace $STEWARD_TEST_CLIENT
```
