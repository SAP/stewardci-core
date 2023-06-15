# Secrets examples

To run these examples you need to set the environment variable `CONTENT_NAMESPACE` accordingly.

## Basic examples

You can find some basic examples in [secretExamples.yaml](secretExamples.yaml).

## Missing secrets

If a secret is referenced in a pipeline run spec but does not exist, the pipeline run will fail with result `error_content`.

```bash
spr=$(kubectl -n "$CONTENT_NAMESPACE" create -f pipelinerun_missing_secret.yaml -o name)
kubectl  -n "$CONTENT_NAMESPACE" get "$spr" -o wide
```

```
NAME                  STARTED   FINISHED   STATUS     RESULT          MESSAGE
missingsecret-rsww9   12s                  finished   error_content   ERROR: preparing failed ...
```

## Secret renaming

It is possible to [rename secrets](../secrets/Secrets.md).
This can be tested with the pipeline run in file [`pipelinerun_secret_rename.yaml`](pipelinerun_secret_rename.yaml).
As a preparation you need to create the secret with the rename annotation.
You can find the renamed secret in the run namespace as listed below.

```bash
kubectl -n "$CONTENT_NAMESPACE" create -f secret_rename.yaml
spr=$(kubectl -n "$CONTENT_NAMESPACE" create -f pipelinerun_secret_rename.yaml -o name)
runnamespace=$(kubectl  -n "$CONTENT_NAMESPACE" get "$spr" -o jsonpath='{.status.namespace}')
kubectl -n "$runnamespace" get secret
```

```
NAME                  TYPE                                  DATA   AGE
default-token-rfs4k   kubernetes.io/service-account-token   3      24s
renamed               kubernetes.io/basic-auth              2      24s
```
