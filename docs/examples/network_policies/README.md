# Network policy examples

To run this examples you need to [create a Tenant](../README.md#tenant) and [set the environment variable](../README.md#pipelinerun) `TENANT_NAMESPACE` accordingly.

## Valid policy configuration

To run this examples apply the `policies.yaml` to your setward namespace.

```
kubectl -n steward-system apply -f policies.yaml
```

### Network policy with internet access

This example pipeline run uses a network policy which allows internet access.

```
kubectl -n  $TENANT_NAMESPACE apply -f pipelinerun_network_internet.yaml
```

The result is a successful run.
```
NAME                             STARTED   FINISHED   STATUS     RESULT          MESSAGE
network-profile-internet-qgqvf   **        **         finished   success         Pipeline completed with result: SUCCESS
```

### Network policy with no network access

This example pipeline run uses a network policy which allows no network access.

```
kubectl -n  $TENANT_NAMESPACE apply -f pipelinerun_network_blocked.yaml
```

The result is a run failing with a content error.

```
NAME                             STARTED   FINISHED   STATUS     RESULT          MESSAGE
network-profile-blocked-qgjfp    **        **         finished   error_content   Command ['git' 'clone' 'https://github.com/SAP-samples/stewardci-example-pipelines' '.'] failed w...
```

### Network policy with unknown policy

This exaple pipeline run tries to use a non existing network policy.

```
kubectl -n  $TENANT_NAMESPACE apply -f pipelinerun_network_unknown.yaml
```

The result is a run failing with a config error.

```
NAME                             STARTED   FINISHED   STATUS     RESULT          MESSAGE
network-profile-unknown-9p8v8    **        **         finished   error_config    ERROR: preparing failed ...
```

## Invalid policy configuration

If the definition of the network policy config map is inconsistent (e.g. not existing default policy), you will get an infrastructure error.

To run this example apply the `policies_inconsistent.yaml` to your setward namespace.

```
kubectl -n steward-system apply -f policies_inconsistent.yaml
kubectl -n  $TENANT_NAMESPACE apply -f pipelinerun_network_internet.yaml
```

Result

```
NAME                             STARTED   FINISHED   STATUS     RESULT          MESSAGE
network-profile-internet-js64h   **        **         finished   error_infra     ERROR: failed to load configuration for pipeline runs ...
```