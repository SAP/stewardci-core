# Steward-Backend API

:warning: **The current API is preliminary and will be changed!**

The Steward backend API is based on Kubernetes resources. Clients use the Kubernetes API/CLI to create and manage them.

Each (frontend) client connecting to the Steward backend gets its own _client namespace_.

Inside its _client namespace_ the client creates `tenant` resources for each tenant it serves. Steward will prepare a separate _tenant namespace_ for each tenant (resource).

Inside a _tenant namespace_ the client creates `pipelinerun` resources for each pipeline execution. Steward will then create a sandbox namespace for each pipelinerun and start a Jenkinsfile runner pod which executes the pipeline.


## Tenant Resource

### Create

A simple `Tenant` resource example can be found in [docs/examples/tenant.yaml](../examples/tenant.yaml).

| Parameter | Description |
| --------- | ----------- |
|`metadata.name`     | the resource name has to be the unique tenant ID |

```bash
$ kubectl apply -f tenant.yaml
```

### Read

The tenant resource in Kubernetes is enriched with a `status` while it is processed and when tenant preparation finished.

```bash
$ kubectl -n <steward-client1> get tenant <tenantId> -oyaml
```
_(shortened example yaml)_
```yaml
status:
  message: Tenant namespace successfully prepared
  progress: Finished
  result: success
  tenantNamespaceName: stu-tn-cl1-test-tenant-09a530
```

| Parameter | Description |
| --------- | ----------- |
|`status.message` | A message describing the latest status |
|`status.progress` | The current progress of processing the tenant resource **(deprecated)**. Possible values:<br>`['', 'InProcess', 'CreateNamespace', 'GetServiceAccount', 'AddRoleBinding', 'Finalize', 'Finished']` |
|`status.result` | The result of the resource processing. Possible values:<br>`['', 'success', 'error_infra', 'error_content']` |
|`status.tenantNamespaceName` | The name of the namespace to be used for this tenant |

:warning: The `status` section is about to change! There will be a `Ready` condition (like for [pods][k8s_pod_conditions] or [nodes][k8s_node_conditions] replacing `message`, `progress` and `result`.

### Delete

When a `Tenant` resource is deleted the corresponding namespace and all linked resources are deleted automatically.


## PipelineRun Resource

### Create

A simple `PipelineRun` resource example can be found in [docs/examples/pipelinerun_ok.yaml](../examples/pipelinerun_ok.yaml). A more complex `PipelineRun` is [docs/examples/pipelinerun_gitscm.yaml](../examples/pipelinerun_gitscm.yaml).

| Parameter | Description |
| --------- | ----------- |
| `spec.jenkinsFile.repoUrl` | the git repository containing the Jenkinsfile to be executed |
| `spec.jenkinsFile.revision` | the branch/revision containing the Jenkinsfile to be executed |
| `spec.jenkinsFile.relativePath` | the relative path to the Jenkinsfile inside the git repository + revision |
| `spec.args` | The arguments specified here will be made available to the pipeline execution |
| `spec.secrets[]` | The secrets specified here will be made available to the pipeline execution. Here you find [more information about secrets](../secrets/Secrets.md) |
| `spec.logging.elasticsearch` | The configuration for pipeline logging to Elasticsearch. If not specified, logging to Elasticsearch is disabled and the default Jenkins log implementation is used (stdout of Jenkinsfile Runner container). |
| `spec.logging.elasticsearch.runID` | The JSON value that should be set as field `runId` in each log entry. It can be any JSON value (`null`, boolean, number, string, list, map). |
| `spec.runDetails.jobName` | The name of the job this pipeline run belongs to. It is used as the name of the Jenkins job and therefore must be a valid Jenkins job name. If null or empty, `job` will be used. |
| `spec.runDetails.sequenceNumber` | The sequence number of the pipeline run, which translates into the build number of the Jenkins job.  If null or empty, `1` is used. |
| `spec.runDetails.cause` | A textual description of the cause of this pipeline run. Will be set as cause of the Jenkins job. If null or empty, no cause information will be available. |


```bash
$ kubectl create -f pipelinerun.yaml
```

### Read

A pipeline resource in Kubernetes is enriched with a `status` while the pipeline is running and when it finished.

```bash
$ kubectl -n <steward-client1-tenant1> get pipelinerun <runName> -oyaml
```
_(shortened example yaml)_
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

| Parameter | Description |
| --------- | ----------- |
|`status.message` | A message describing the latest status |
|`status.result`  | The result of the pipeline run. Possible values:<br>`['success', 'error_infra', 'error_content', 'killed', 'timeout']` |
|`status.state`   | The current state of the pipeline run. Possible values:<br>`['', 'preparing', 'waiting', 'running', 'cleaning', 'finished']` |
|`status.stateDetails` | Details of the latest state, like start time and finish time |
|`status.stateHistory` | The history of all state (changes) including details like start time and finish time |

:warning: The `status` section is about to change! There will be conditions (like for [pods][k8s_pod_conditions] or [nodes][k8s_node_conditions] replacing `state`, `result` and `message`. The fields `container`, `logUrl`, `stateDetails` and `stateHistory` will possibly be removed.

### Delete

The sandbox namespace of a PipelineRun is deleted immediately once the pipeline finished &ndash; no need to delete the PipelineRun resource. Still PipelineRun resources can be deleted once they are not needed anymore.


[k8s_pod_conditions]: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions
[k8s_node_conditions]: https://kubernetes.io/docs/concepts/architecture/nodes/#condition
