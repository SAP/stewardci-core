# Integration tests

## Preparation

### Prepare test client

```bash
export STEWARD_TEST_CLIENT=$(kubectl apply -f test-client.yaml -o=name)
export STEWARD_TEST_CLIENT=${STEWARD_TEST_CLIENT#*/}
```

### Prepare test tenant
This setup is optional. If no test tenant is created it will be created automatically by the test and cleaned up after the test completed.

If you want to keep the tenant after the test prepare one manually and clean it up manually after the tests.
```bash
export TENANT_NAME=$(kubectl -n $STEWARD_TEST_CLIENT create -f test-tenant.yaml -o=name)
export TENANT_NAME=${TENANT_NAME#*/}
# wait until tenant namespace is created
export STEWARD_TEST_TENANT=$(kubectl -n $STEWARD_TEST_CLIENT get tenants.steward.sap.com ${TENANT_NAME} -o=jsonpath={.status.tenantNamespaceName})
echo $STEWARD_TEST_TENANT
```

### Run framework tests to check if the test framework works correctly

```bash
cd framework
go test -count=1 -tags=frameworktest -v --kubeconfig $KUBECONFIG .
```

## Run tests

### Integration tests

```bash
cd integrationtest
go test -count=1 -tags=e2e -v --kubeconfig $KUBECONFIG .
```

### Load tests

```bash
cd loadtest
go test -count=1 -tags=loadtest -v --kubeconfig $KUBECONFIG .
```

## Cleanup
```bash
kubectl delete namespace $STEWARD_TEST_CLIENT
```
