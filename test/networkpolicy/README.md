# Integration tests for network policies

You need to be in the directory of this README to execute this tests.

## Preparation

```bash
# create client
export STEWARD_TEST_CLIENT=$(kubectl apply -f test-client.yaml -o=name)
export STEWARD_TEST_CLIENT=${STEWARD_TEST_CLIENT#*/}


STEWARD_NAMESPACE=steward-system

# backup current network policy
kubectl -n ${STEWARD_NAMESPACE} get cm steward-pipelineruns -oyaml > backup_network_policy.yaml

# start netcat server
kubectl apply -f netcat.yaml

```

## Run tests

```bash
## Prepare
STEWARD_NAMESPACE=steward-system

# backup current network policy
kubectl -n ${STEWARD_NAMESPACE} get cm steward-pipelineruns -oyaml > backup_network_policy.yaml

# start netcat server
kubectl apply -f netcat.yaml

## test with network policy in place
go test -count=1 -tags=closednet -v --kubeconfig $KUBECONFIG .

## open network policy to allow access to netcat server
kubectl -n ${STEWARD_NAMESPACE} apply -f open_policy.yaml

## test successfull access to netcat server
go test -count=1 -tags=opennet -v --kubeconfig $KUBECONFIG .

```

## Cleanup

```bash
# delete netcat server
kubectl delete -f netcat.yaml

# restore network policy
grep -v "^  uid" backup_network_policy.yaml | grep -v  "^  resourceVersion" | kubectl apply -f -

# delete client
kubectl delete namespace $STEWARD_TEST_CLIENT
```
