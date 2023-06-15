# Integration tests for network policies

You need to be in the directory of this README to execute this tests.

## Preparation

```bash
STEWARD_TEST_NAMESPACE=$(kubectl create -f - -o name <<<'{ "apiVersion": "v1", "kind": "Namespace", "metadata": { "generateName": "steward-test-networkpolicies-" } }') && \
export STEWARD_TEST_NAMESPACE=${STEWARD_TEST_NAMESPACE#*/}
```

## Run tests

Depending on the hyperscaler you need to use diffent policies to allow the connection.

### Test on AWS

```bash
cp open_policy{_aws,}.yaml
./run_test.sh
```

### Test on GCP

```bash
cp open_policy{_gcp,}.yaml
./run_test.sh
```


## Cleanup

```bash
# delete client
kubectl delete namespace "$STEWARD_TEST_NAMESPACE"
```
