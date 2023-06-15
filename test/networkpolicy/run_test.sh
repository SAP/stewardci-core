#!/bin/bash
set -eu -o pipefail

STEWARD_NAMESPACE=steward-system
STEWARD_NETWORK_POLICY_CONFIGMAP="steward-pipelineruns-network-policies"

POLICY_BACKUP_FILE=backup_network_policy_conf.yaml
OPEN_POLICY_FILE=open_policy.yaml
NETCAT_DEPLOYMENT_CONFIG_FILE=netcat.yaml

echo "Backup network policy"
kubectl -n "${STEWARD_NAMESPACE}" get configmap "$STEWARD_NETWORK_POLICY_CONFIGMAP" -o yaml >"$POLICY_BACKUP_FILE"

echo "Start netcat server"
kubectl apply -f "$NETCAT_DEPLOYMENT_CONFIG_FILE"

echo "--- 1st test with original network policy ---"
go test -count=1 -tags=closednet -v --kubeconfig "$KUBECONFIG" . || true
echo "--- END 1st test ---"

echo "Open network policy to allow access to netcat server"
kubectl -n "${STEWARD_NAMESPACE}" delete configmap "$STEWARD_NETWORK_POLICY_CONFIGMAP"
kubectl -n "${STEWARD_NAMESPACE}" create configmap "$STEWARD_NETWORK_POLICY_CONFIGMAP" \
    --from-literal=_default=default \
    --from-file=default="$OPEN_POLICY_FILE"

echo "--- 2nd test with new network policy ---"
go test -count=1 -tags=opennet -v --kubeconfig "$KUBECONFIG" . || true
echo "--- END 2nd test ---"

echo "Delete netcat server"
kubectl delete -f "$NETCAT_DEPLOYMENT_CONFIG_FILE"

echo "Restore original network policy"
grep -v "^  uid" "$POLICY_BACKUP_FILE" | grep -v  "^  resourceVersion" | kubectl apply -f -
echo "Finish"
