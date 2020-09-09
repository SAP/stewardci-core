# How to test permissions

Prerequisites: steward-system and a steward-client called `steward-client-1` prepared.

## Test `client namespace` with `client service account`

Allowed permissions:

```sh
cd ../examples

# List tenants
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n steward-client-1 get tenants

# Create tenant
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n steward-client-1 create -f tenant.yaml

# Get tenant
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n steward-client-1 get tenant -oyaml

# Delete tenant
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n steward-client-1 delete tenant 4e93d9d5-276e-47ca-a570-b3a763aaef3e

```

Forbidden permissions:

```sh
cd ../examples

# Get other resources
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n steward-client-1 get pods

# Get pipeline runs in own namespace (only works in tenant namespace)
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n steward-client-1 get pipelineruns.steward.sap.com

# Get secrets in own namespace (only works in tenant namespace)
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n steward-client-1 get secrets

# Access other namespaces
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n default get tenant

```

## Test `tenant namespace` with `client service account`


Allowed permissions:
```sh
cd ../examples

# Create pipeline run
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n stu-tn1-4e93d9d5-276e-47ca-a570-b3a763aaef3e-e72657 create -f pipelinerun_ok.yaml

# List pipeline runs
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n stu-tn1-4e93d9d5-276e-47ca-a570-b3a763aaef3e-e72657 get pipelineruns.steward.sap.com

# Get pipeline run
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n stu-tn1-4e93d9d5-276e-47ca-a570-b3a763aaef3e-e72657 get pipelinerun.steward.sap.com ok-fzfwk -oyaml

# Delete pipeline run
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n stu-tn1-4e93d9d5-276e-47ca-a570-b3a763aaef3e-e72657 delete pipelinerun.steward.sap.com ok-fzfwk


# Get secrets
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n stu-tn1-4e93d9d5-276e-47ca-a570-b3a763aaef3e-e72657 get secrets


```

Forbidden permissions:
```sh
cd ../examples

# Create tenant in tenant namespace (only for pipeline runs)
kubectl --as=system:serviceaccount:steward-client-1:steward-client -n stu-tn1-4e93d9d5-276e-47ca-a570-b3a763aaef3e-e72657 create -f tenant.yaml

```
