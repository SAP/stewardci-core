# Steward-Backend API

:warning: **The current API is preliminary and will be changed!**

The Steward backend API is based on Kubernetes resources. Clients use the Kubernetes API/CLI to create and manage them.

Each (frontend) client connecting to the Steward backend gets its own _client namespace_.

Inside its _client namespace_ the client creates Tenant resources for each of its own tenants. Steward will prepare a separate _tenant namespace_ for each tenant (resource).

Inside a _tenant namespace_ the client creates PipelineRun resources for each pipeline execution. Steward will then create a sandbox namespace for each pipeline run and start a Jenkinsfile runner pod which executes the pipeline.


## Tenant Resource

### Spec

#### Examples

A simple Tenant resource example can be found in [docs/examples/tenant.yaml](../examples/tenant.yaml).


#### Fields

| Field | Description |
| --------- | ----------- |
| `apiVersion` | `steward.sap.com/v1alpha1` |
| `kind` | `Tenant` |
| `metadata.name` | The resource name has to be the unique tenant ID. |


### Status

The `status` section of the Tenant resources lets clients know about the tenant namespace assigned exclusively to a tenant.

After a client created a __new Tenant resource__, the Steward controller tries to create the hereby requested state, which is this:

- A tenant namespace exists that is exclusively assigned to this tenant.

- Service account `<tenant_namespace>::default` (where `<tenant_namespace>` is the name of the namespace assigned exlusively to the tenant) has the permissions needed to manage further resources in the tenant namespace.

- Service account `<client_namespace>::default` (where `<client_namespace>` is the namespace where the `Tenant` resource belongs to) has the permissions needed to manage further resources in the tenant namespace.

Once the controller has finished the initialization successfully, field `status.tenantNamespaceName` will be set and will not change anymore during the lifetime of the Tenant resource object.
Note that Steward does _not_ give any guarantees on how long the initialization takes.
Clients must watch or poll the resource object until field `status.tenantNamespaceName` is set, before using the tenant namespace.

The Steward controller periodically checks the actual state of all __existing Tenant resources__ and tries to change it to the desired state if there are deviations (reconciliation):

- The role binding in the tenant namespace gets updated/recreated if needed, for instance if the client namespace's annotation `steward.sap.com/tenant-role` (defining the RBAC role to be assigned to the above-mentioned service accounts) has changed or the role binding does not exist anymore.

- If `status.tenantNamespaceName` refers to a namespace that does not exist anymore, the reconciliation fails and the status is set accordingly (see below).
  As this never happens under normal circumstances and probably means that data has been lost, the tenant namespace will not be recreated automatically.
  A Steward operator may resolve the issue by restoring the tenant namespace with all its former contents from a backup.

In case the __initialization or reconciliation fails__, the Steward controller sets the _ready condition's_ status to `False` to indicate that the Tenant is not ready for use (the ready condition is explained below).

Steward operators should monitor the status of all Tenant resource objects and react on:

- Uninitialized resource objects older than a certain threshold, e.g. 10 seconds.
- Resource objects with the ready condition being `False` for longer than a certain threshold, e.g. 10 seconds.

Clients are _not_ required to check the status of the Tenant resource object before each operation they perform in the respective tenant namespace.
Instead they should try the operations (Kubernetes API calls) and check the response for errors.


### Examples

The initialization was successful:

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

The initialization has failed because the tenant namespace could not be created:

```yaml
apiVersion: steward.sap.com/v1apha1
kind: Tenant
metadata:
  name: tenant1
  namespace: steward-c-client1
status:
  conditions:
  - type: Ready
    status: "False"
    reason: Failed
    message: |
      Failed to create the tenant namespace.
    lastTransitionTime: "2019-11-02T07:35:16Z"
  # no tenantNamespaceName set
```

The reconciliation failed because the tenant namespace does not exist anymore:

```yaml
apiVersion: steward.sap.com/v1apha1
kind: Tenant
metadata:
  name: tenant1
  namespace: steward-c-client1
status:
  conditions:
  - type: Ready
    status: "False"
    reason: InvalidDependentResource
    message: |
      The tenant namespace "steward-t-client1-tenant1-83a4cf" does not exist anymore.
      This issue must be analyzed and fixed by an operator.
    lastTransitionTime: "2019-11-02T07:35:16Z"
  tenantNamespaceName: steward-t-client1-tenant1-83a4cf
```


#### Fields

| Field | Description |
| --------- | ----------- |
| `status.conditions` | (array,optional) A list of condition objects describing the lastest observed state of the resource. For each type of condition at most one entry exists. Omitting this field is equivalent to specifying it with an empty array value. See [_Conditions_](#conditions) below for a specification of condition types defined for Tenant resources. |
| `status.conditions[*].type` | (string) The type of the condition as a unique, one-word, lower-case string. |
| `status.conditions[*].status` | (string,optional) The status of the condition with one of the values `True`, `False` and `Unknown`. Any condition not listed in `status.conditions` must be treated as if it has status `Unknown`. |
| `status.conditions[*].reason` | (string,optional) A unique, one-word, camel-case reason for the condition's last transition. |
| `status.conditions[*].message` | (string,optional) A human-readable message indicating the details of the condition's last transition. |
| `status.conditions[*].lastTransitionTime` | (time,optional) The time of the condition's last transition. |
| `status.tenantNamespaceName` | (string,optional) The name of the namespace assigned exclusively to this tenant. As long as the Tenant resource is not successfully initialized, this field is not set. |


#### Conditions

The Kubernetes API conventions [recommends conditions][k8s_api_conventions_conditions] as the means to communicate the latest observed state of a resource.
Conditions are used by many Kubernetes resource types.

##### Ready Condition

The condition of type `ready` is the main condition of a Tenant resource (and currently the only one, which might change in the future).

If the condition's status is `True` the resource's `status.tenantNamespaceName` is guaranteed to be set and the tenant namespace was correctly set up last time the Steward controller verified the resource state.
Note that since then the state might have changed again but not yet been recognized by the Steward controller.
Fields `reason` and `message` are not specified if `status` is `True`.

If the condition's status in `False`, fields `reason` and `message` will be set.
Possible values of `reason` are:

- `Failed`: Indicates that the reason for the status is an unspecified failure.
- `InvalidDependentResource`: Indicates that the reason for the status is the state of another resource controlled by this resource, e.g. the tenant namespace or the role binding in the tenant namespace.

Consumers of the resource status should not strongly rely on the value of the `reason` field, as the set of possible values might change in future versions of Steward without considering this as incompatibility.
The `reason` and `message` fields have informative character only.
If consumers must rely on detailed information about the status of non-ready resource objects, new condition types must be introduced with a new Steward versions.

Field `lastTransitionTime` is always set, except when the condition is not specified in the resource status at all (which for instance is the case for newly created resource objects).


### Deletion

When a Tenant resource is deleted the assigned namespace will be deleted automatically, including all resources within that namespace.


## PipelineRun Resource

### Spec

#### Examples

A simple PipelineRun resource example can be found in [docs/examples/pipelinerun_ok.yaml](../examples/pipelinerun_ok.yaml). A more complex PipelineRun is [docs/examples/pipelinerun_gitscm.yaml](../examples/pipelinerun_gitscm.yaml).


#### Fields

| Field | Description |
| --------- | ----------- |
| `apiVersion` | `steward.sap.com/v1alpha1` |
| `kind` | `PipelineRun` |
| `spec.intent` | (string,optional) The intention of the client regarding the way this pipeline run should be processed. The value `run` indicates that the pipeline should run to completion, while the value `abort` indicates that the pipeline processing should be stopped as soon as possible. Omitting the field  or specifying an empty string value is equivalent to value `run`. |
| `spec.jenkinsFile` | (object,mandatory) The configuration of the Jenkins pipeline definition to be executed. |
| `spec.jenkinsFile.repoUrl` | (string,mandatory) The URL of the Git repository containing the pipeline definition (aka `Jenkinsfile`). |
| `spec.jenkinsFile.revision` | (string,mandatory) The revision of the pipeline Git repository to used, e.g. `master`. |
| `spec.jenkinsFile.relativePath` | (string,mandatory) The relative pathname of the pipeline definition file in the repository check-out, typically `Jenkinsfile`. |
| `spec.jenkinsFile.repoAuthSecret` | (string,optional) The name of the Kubernetes `v1/Secret` resource object of type `kubernetes.io/basic-auth` that contains the username and password for authentication when cloning from `spec.jenkinsFile.repoUrl`. See [docs/secrets/Secrets.md](../secrets/Secrets.md) for details. |
| `spec.args` | (object,optional) The parameters to pass to the pipeline, as key-value pairs of type string. |
| `spec.secrets` | (array of string,optional) The list of secrets to be made available to the pipeline execution. Each entry in the list is the name of a Kubernetes `v1/Secret` resource object in the same namespace as the PipelineRun object itself. See [docs/secrets/Secrets.md](../secrets/Secrets.md) for details. |
| `spec.imagePullSecrets` | (array of string,optional) The list of image pull secrets required by the pipeline run to pull images of custom containers from private registries. Each entry in the list is the name of a Kubernetes `v1/Secret` resource object of type `kubernetes.io/dockerconfigjson` in the same namespace as the PipelineRun object itself. See [docs/secrets/Secrets.md](../secrets/Secrets.md) for details. |
| `spec.jenkinsfileRunner` | (object, optional) Configuration of the Jenkinsfile Runner container (see below). |
| `spec.jenkinsfileRunner.image` | (string, optional) The Jenkinsfile Runner container image to be used for this pipeline run. If not specified, a default image configured for the Steward installation will be used.<br/><br/>Example: `my-org/my-jenkinsfile-runner:latest` |
| `spec.jenkinsfileRunner.imagePullPolicy` | (string, optional) The image pull policy for `spec.jenkinsfileRunner.image`. It applies only if `spec.jenkinsfileRunner.image` is set, i.e. it does _not_ overwrite the image pull policy of the _default_ Jenkinsfile Runner image. Defaults to 'IfNotPresent'.<br/><br/>**Currently broken, `IfNotPresent` is used in any case. See [tektoncd/pipeline #3423](https://github.com/tektoncd/pipeline/issues/3423)** |
| `spec.runDetails` | (object,optional) Properties of the Jenkins build object. |
| `spec.runDetails.jobName` | (string,optional) The name of the job this pipeline run belongs to. It is used as the name of the Jenkins job and therefore must be a valid Jenkins job name. If null or empty, `job` will be used. |
| `spec.runDetails.sequenceNumber` | (string,optional) The sequence number of the pipeline run, which translates into the build number of the Jenkins job.  If null or empty, `1` is used. |
| `spec.runDetails.cause` | (string,optional) A textual description of the cause of this pipeline run. Will be set as cause of the Jenkins job. If null or empty, no cause information will be available. |
| `spec.logging` | (object,optional) The logging configuration. |
| `spec.logging.elasticsearch` | (object,optional) The configuration for pipeline logging to Elasticsearch. If not specified, logging to Elasticsearch is disabled and the default Jenkins log implementation is used (stdout of Jenkinsfile Runner container). |
| `spec.logging.elasticsearch.runID` | (any,optional) The JSON value that should be set as field `runId` in each log entry in Elasticsearch. It can be any JSON value (`null`, boolean, number, string, list, map). |


#### Mutability

All fields except those described below MUST NOT be changed after a PipelineRun resource has been created.

Mutable fields:

- `spec.intent`: The following transitions are allowed:

    - from unspecified to one of {empty string, `run`, `abort`}
    - from empty string to one of {unspecified, `run`, `abort`}
    - from `run` to one of {unspecified, empty string, `abort`}

  All other transitions are prohibited.


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
| `status.startedAt` | (time,optional) The time the pipeline run has been started at. It gets set on start and remains unchanged for the object's remaining lifetime. |
| `status.finishedAt` | (time,optional) The time the pipeline run has been finished at. It gets set when finished (`status.result` is also set) and remains unchanged for the object's remaining lifetime. |
| `status.result` | (string,optional) The result code of the pipeline run as single-word string. Possible values are `success`, `error_infra`, `error_content`, `aborted` and `timeout`. |
| `status.message` | (string,optional) A message describing the reason for the latest status. May not be set or an empty string in case no message is provided. |
| `status.state` | (string,optional) The name of the current state in the pipeline run process as a single-word string. Possible values are `new`, `preparing`, `waiting`, `running`, `cleaning` and `finished`. An omitted field,`null` value or an empty string value is equivalent to `new`. |
| `status.stateDetails` | (object,optional) Details of the current state (`status.state`). It is set if `status.state` is set. |
| `status.stateDetails.state` | (string,mandatory) The name of the state in the pipeline run process as a single-word string. See `status.state`. |
| `status.stateDetails.startedAt` | (time,mandatory) The time the state has been entered. |
| `status.stateDetails.finishedAt` | (time,optional) The time the state has been left. It is not set (omitted or `null` value) as long as the state has not been left. |
| `status.stateHistory` | (array,optional) The history of states the pipeline run process has had so far. The elements are objects of the same structure as `status.stateDetails`. |

:warning: The `status` section is about to change! There will be conditions (like for [pods][k8s_pod_conditions] or [nodes][k8s_node_conditions] replacing `state`, `result` and `message`. The fields `container`, `logUrl`, `stateDetails` and `stateHistory` will possibly be removed.


### Deletion

Steward currently does not delete PipelineRun resources automatically. It is the clients' responsibility to delete them when they are no longer needed, reached a certain age or whatever the deletion criterion is.

The sandbox namespace of a PipelineRun gets deleted immediately after the pipeline run has finished &ndash; no need to delete the PipelineRun resource itself to clean up.


## Links

- [Kubernetes Design Principles][k8s_design_principles]
- [Kubernetes API conventions][k8s_api_conventions]



[k8s_pod_conditions]: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions
[k8s_node_conditions]: https://kubernetes.io/docs/concepts/architecture/nodes/#condition
[k8s_api_conventions]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md
[k8s_api_conventions_conditions]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
[k8s_design_principles]: https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/principles.md
