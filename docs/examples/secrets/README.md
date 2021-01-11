# Secrets examples

To run this examples you need to [create a Tenant](../README.md#tenant) and [set the environment variable](../README.md#pipelinerun) `TENANT_NAMESPACE` accordingly.

## Basic examples

You can find some basic examples in [secretExamples.yaml](secretExamples.yaml).

## Missing secrets

If a secret is missing the pipelien run will fail with an content error.

```bash
spr=$(kubectl -n "$TENANT_NAMESPACE" create -f pipelinerun_missing_secret.yaml -oname)
kubectl  -n "$TENANT_NAMESPACE" get $spr -owide
```

```
NAME                  STARTED   FINISHED   STATUS     RESULT          MESSAGE
missingsecret-rsww9   12s                  finished   error_content   ERROR: preparing failed ...
```

## Secret renaming

It is possible to [rename secrets](../secrets/Secrets.md). This can be tested with the pipeline run in the file [pipelinerun_secret_rename.yaml](pipelinerun_secret_rename.yaml). As preparation you need to create the secret with the rename annotation. You can find the renamed secret in the run namespace as listed below.

```bash
kubectl -n "$TENANT_NAMESPACE" create -f secret_rename.yaml
spr=$(kubectl -n "$TENANT_NAMESPACE" create -f pipelinerun_secret_rename.yaml -oname)
runnamespace=$(kubectl  -n "$TENANT_NAMESPACE" get $spr -ojsonpath='{.status.namespace}')
kubectl -n "$runnamespace" get secret
```

```
NAME                  TYPE                                  DATA   AGE
default-token-rfs4k   kubernetes.io/service-account-token   3      24s
renamed               kubernetes.io/basic-auth              2      24s
```