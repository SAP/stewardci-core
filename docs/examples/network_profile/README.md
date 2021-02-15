# Network profile examples

To run these examples you need to [create a Tenant](../README.md#tenant) and [set the environment variable](../README.md#pipelinerun) `TENANT_NAMESPACE` accordingly.


## Valid profile configuration

To run these examples apply `profiles.yaml` to your Steward system namespace, e.g. `steward-system`:

```
kubectl -n steward-system apply -f profiles.yaml
```

### Network profile with internet access

This example pipeline run uses a network profile which allows internet access.

```
kubectl -n  $TENANT_NAMESPACE apply -f pipelinerun_network_internet.yaml
```

The result is a successful run.
```
NAME                             STARTED   FINISHED   STATUS     RESULT          MESSAGE
network-profile-internet-qgqvf   **        **         finished   success         Pipeline completed with result: SUCCESS
```

### Network profile without network access


This example pipeline run uses a network profile which blocks any network communication.


```
kubectl -n  $TENANT_NAMESPACE apply -f pipelinerun_network_blocked.yaml
```

The result is a run failing with a content error.

```
NAME                             STARTED   FINISHED   STATUS     RESULT          MESSAGE
network-profile-blocked-qgjfp    **        **         finished   error_content   Command ['git' 'clone' 'https://github.com/SAP-samples/stewardci-example-pipelines' '.'] failed w...
```

### Unknown network profile


This example pipeline run tries to use a nonexistent network profile.


```
kubectl -n  $TENANT_NAMESPACE apply -f pipelinerun_network_unknown.yaml
```

The pipeline run result is `error_config`, which indicates that the pipeline run specification was erroneous.

```
NAME                             STARTED   FINISHED   STATUS     RESULT          MESSAGE
network-profile-unknown-9p8v8    **        **         finished   error_config    ERROR: preparing failed ...
```

## Invalid profile configuration

If the definition of the network profile config map is inconsistent (e.g. not existing default profile), all pipeline runs will fail with result `error_infra`.


To run this example apply `profiles_inconsistent.yaml` to your Steward system namespace, e.g.:

```
kubectl -n steward-system apply -f profiles_inconsistent.yaml
kubectl -n  "$TENANT_NAMESPACE" apply -f pipelinerun_network_internet.yaml
```


Result

```
NAME                             STARTED   FINISHED   STATUS     RESULT          MESSAGE
network-profile-internet-js64h   **        **         finished   error_infra     ERROR: failed to load configuration for pipeline runs ...
```
