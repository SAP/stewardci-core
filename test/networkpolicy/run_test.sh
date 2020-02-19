#!/bin/bash +x

STEWARD_NAMESPACE=steward-system

echo "Backup network policy"
kubectl -n ${STEWARD_NAMESPACE} get cm steward-pipelineruns -oyaml > backup_network_policy.yaml

echo "Start netcat server"
kubectl apply -f netcat.yaml

echo "--- 1st test with original network police ---"
go test -count=1 -tags=closednet -v --kubeconfig $KUBECONFIG .
echo "--- END 1st test ---"

echo "open network policy to allow access to netcat server"
kubectl -n ${STEWARD_NAMESPACE} apply -f open_policy.yaml

echo "--- 2nd test with new network policy ---"
go test -count=1 -tags=opennet -v --kubeconfig $KUBECONFIG .
echo "--- END 2nd test ---"

echo "delete netcat server"
kubectl delete -f netcat.yaml

echo "restore original network policy"
grep -v "^  uid" backup_network_policy.yaml | grep -v  "^  resourceVersion" | kubectl apply -f -
echo "Finish"
