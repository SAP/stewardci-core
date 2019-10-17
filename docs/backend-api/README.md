# Steward-Backend API

:warning: **The current API is preliminary and will be changed!**

The Steward backend API is based on Kubernetes resources. Clients use the Kubernetes API/CLI to create and manage them.

Each (frontend) client connecting to the Steward backend gets its own _client namespace_.

Inside its _client namespace_ the client creates `tenant` resources for each tenant it serves. Steward will prepare a separate _tenant namespace_ for each tenant (resource).

Inside a _tenant namespace_ the client creates `pipelinerun` resources for each pipeline execution. Steward will then create a sandbox namespace for each pipelinerun and start a Jenkinsfile runner pod which executes the pipeline.


## Tenant Resource

### Spec

#### Examples

A simple `Tenant` resource example can be found in [docs/examples/tenant.yaml](../examples/tenant.yaml).


#### Fields

| Field | Description |
| --------- | ----------- |
| `apiVersion` | `steward.sap.com/v1alpha1` |
| `kind` | `Tenant` |
| `metadata.name` | The resource name has to be the unique tenant ID. |


### Status

The `status` section of the `Tenant` resources lets clients know about the tenant namespace assigned exclusively to a tenant.

After a client created a new `Tenant` resource, the Steward controller tries to achieve the hereby requested state:

- A tenant namespace exists that is exclusively assigned to this tenant.

- Service account `<tenant_namespace>::default` (where `<tenant_namespace>` is the name of the namespace assigned exlusively to the tenant) has the permissions needed to manage further resources in the tenant namespace.

- Service account `<client_namespace>::default` (where `<client_namespace>` is the namespace where the `Tenant` resource belongs to) has the permissions needed to manage further resources in the tenant namespace.

The Steward controller periodically checks the actual state of the resource and tries to change it to the desired state:

- The role binding in the tenant namespace gets updated/recreated if needed, for instance if the client namespace's annotation `steward.sap.com/tenant-role` (defining the RBAC role to be assigned to the above-mentioned service accounts) has changed.

- If `status.tenantNamespaceName` refers to a namespace that does not exist anymore, the ready condition is set to `False` indicating that the tenant is no longer ready to be used. As this never happens under normal circumstances and probably means that data has been lost, the tenant namespace will not be recreated automatically. Operators should monitor tenants, and must  analyze and fix the underlying issue if such situations occur.


### Examples

```yaml
apiVersion: steward.sap.com/v1apha1
kind: Tenant
metadata:
  name: tenant1
  namespace: steward-c-client1
status:
  conditions:
  - type: Ready
    status: "True"
    lastTransitionTime: "2019-11-01T08:15:36Z"
  tenantNamespaceName: steward-t-client1-tenant1-83a4cf
```


#### Fields

| Field | Description |
| --------- | ----------- |
| `status.conditions` | (array) A list of condition objects describing the lastest observed state of the resource. For each type of condition at most one entry exists. See _Conditions_ below. |
| `status.conditions[*].type` | (string) The type of the condition. See _Conditions_ below. |
| `status.conditions[*].status` | (string,optional) The status of the condition with one of the values `True`, `False` and `Unknown`. A condition that is not listed in `status.conditions` has status `Unknown`. |
| `status.conditions[*].reason` | (string,optional) A unique, one-word, camel-case reason for the condition's last transition. |
| `status.conditions[*].message` | (string,optional) A human-readable message indicating the details of the condition's last transition. |
| `status.conditions[*].lastTransitionTime` | (time) The time of the condition's last transition. |
| `status.tenantNamespaceName` | (string) The name of the namespace assigned exclusively to this tenant. As long as the tenant namespace has not been created successfully, this field is not set. |


#### Conditions

Currently the following types of conditions are defined:

- `ready`: The ready condition is the main condition. If its status is `True`, `status.tenantNamespaceName` is guaranteed to be set and the tenant namespace was correctly set up last time the Steward controller verified the resource state. Note that since then the state might have changed again but not yet been recognized by the Steward controller.


### Deletion

When a `Tenant` resource is deleted the assigned namespace will be deleted automatically, including all resources within that namespace.


## PipelineRun Resource

### Spec

#### Examples

A simple `PipelineRun` resource example can be found in [docs/examples/pipelinerun_ok.yaml](../examples/pipelinerun_ok.yaml). A more complex `PipelineRun` is [docs/examples/pipelinerun_gitscm.yaml](../examples/pipelinerun_gitscm.yaml).


#### Fields

| Field | Description |
| --------- | ----------- |
| `apiVersion` | `steward.sap.com/v1alpha1` |
| `kind` | `PipelineRun` |
| `spec.jenkinsFile` | (object) The configuration of the Jenkins pipeline definition to be executed. |
| `spec.jenkinsFile.repoUrl` | (string,mandatory) The URL of the Git repository containing the pipeline definition. |
| `spec.jenkinsFile.revision` | (string,mandatory) The revision of the pipeline Git repository to used, e.g. `master`. |
| `spec.jenkinsFile.relativePath` | (string,mandatory) The relative pathname of the pipeline definition file in the repository check-out, typically `Jenkinsfile`. |
| `spec.args` | (object,optional) The parameters to pass to the pipeline, as key-value pairs of type string. |
| `spec.secrets` | (array,optional) The names of Kubernets secrets in the same (tenant) namespace to be made available to the pipeline execution. See [docs/secrets/Secrets.md](../secrets/Secrets.md) for details. |
| `spec.logging` | (object,optional) The logging configuration. |
| `spec.logging.elasticsearch` | (object,optional) The configuration for pipeline logging to Elasticsearch. If not specified, logging to Elasticsearch is disabled and the default Jenkins log implementation is used (stdout of Jenkinsfile Runner container). |
| `spec.logging.elasticsearch.runID` | (any,optional) The JSON value that should be set as field `runId` in each log entry in Elasticsearch. It can be any JSON value (`null`, boolean, number, string, list, map). |


### Status

The `status` section informs clients about the progress and result of pipeline runs.


#### Examples

```yaml
status:
  container:
    terminated:
      containerID: docker://1edf960e935fdc60fb3cfb067858287ad272eecf70fb9ab9ef2cb87b48e71011
      exitCode: 0
      finishedAt: "2019-07-29T07:00:51Z"
      message: |
        Pipeline completed with result: SUCCESS
      reason: Completed
      startedAt: "2019-07-29T06:58:20Z"
  logUrl: ""
  message: |
    Pipeline completed with result: SUCCESS
  result: success
  state: finished
  stateDetails:
    finishedAt: null
    startedAt: "2019-07-29T07:01:07Z"
    state: finished
  stateHistory:
  - finishedAt: "2019-07-29T06:58:17Z"
    startedAt: "2019-07-29T06:58:17Z"
    state: preparing
  - finishedAt: "2019-07-29T06:58:18Z"
    startedAt: "2019-07-29T06:58:17Z"
    state: waiting
  - finishedAt: "2019-07-29T07:01:06Z"
    startedAt: "2019-07-29T06:58:18Z"
    state: running
  - finishedAt: "2019-07-29T07:01:07Z"
    startedAt: "2019-07-29T07:01:06Z"
    state: cleaning
  - finishedAt: "2019-07-29T07:01:08Z"
    startedAt: "2019-07-29T07:01:07Z"
    state: finished
```


#### Fields

| Field | Description |
| --------- | ----------- |
|`status.message` | A message describing the latest status |
|`status.result`  | The result of the pipeline run. Possible values:<br>`['success', 'error_infra', 'error_content', 'killed', 'timeout']` |
|`status.state`   | The current state of the pipeline run. Possible values:<br>`['', 'preparing', 'waiting', 'running', 'cleaning', 'finished']` |
|`status.stateDetails` | Details of the latest state, like start time and finish time |
|`status.stateHistory` | The history of all state (changes) including details like start time and finish time |

:warning: The `status` section is about to change! There will be conditions (like for [pods][k8s_pod_conditions] or [nodes][k8s_node_conditions] replacing `state`, `result` and `message`. The fields `container`, `logUrl`, `stateDetails` and `stateHistory` will possibly be removed.


### Deletion

Steward currently does not delete `PipelineRun` resources automatically. It is the clients' responsibility to delete them when they are no longer needed, reached a certain age or whatever the deletion criterion is.

The sandbox namespace of a `PipelineRun` gets deleted immediately after the pipeline run has finished &ndash; no need to delete the PipelineRun resource itself to clean up.




[k8s_pod_conditions]: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions
[k8s_node_conditions]: https://kubernetes.io/docs/concepts/architecture/nodes/#condition
