# Steward-Backend API

:warning: **The current API is preliminary and will be changed!**

The Steward backend API is based on Kubernetes resources. Clients use the Kubernetes API/CLI to create and manage them.

Inside a _content namespace_ the client creates PipelineRun resources for each pipeline execution. Steward will then create a sandbox namespace for each pipeline run and start a Jenkinsfile runner pod which executes the pipeline.

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
| `spec.profiles` | (object, optional) The selection of configuration profiles for various aspects that should be applied for the pipeline run (see below). |
| `spec.profiles.network` | (string, optional) The name of the network profile to be used for the pipeline run.<br/><br/>Network profiles currently define the network policy for the pipeline run sandbox. In the future this might be extended to other network-related settings.<br/><br/>Network profiles are configured for each Steward installation individually. Ask the Steward administrator for possible values. For vanilla Steward installations there's one network profile called `default`.<br/><br/>If not set or empty, a default network profile will be used. |
| `spec.jenkinsfileRunner` | (object, optional) Configuration of the Jenkinsfile Runner container (see below). |
| `spec.jenkinsfileRunner.image` | (string, optional) The Jenkinsfile Runner container image to be used for this pipeline run. If not specified, a default image configured for the Steward installation will be used.<br/><br/>Example: `my-org/my-jenkinsfile-runner:latest` |
| `spec.jenkinsfileRunner.imagePullPolicy` | (string, optional) The image pull policy for `spec.jenkinsfileRunner.image`. It applies only if `spec.jenkinsfileRunner.image` is set, i.e. it does _not_ overwrite the image pull policy of the _default_ Jenkinsfile Runner image. Defaults to 'IfNotPresent'. |
| `spec.runDetails` | (object,optional) Properties of the Jenkins build object. |
| `spec.runDetails.jobName` | (string,optional) The name of the job this pipeline run belongs to. It is used as the name of the Jenkins job and therefore must be a valid Jenkins job name. If null or empty, `job` will be used. |
| `spec.runDetails.sequenceNumber` | (string,optional) The sequence number of the pipeline run, which translates into the build number of the Jenkins job.  If null or empty, `1` is used. |
| `spec.runDetails.cause` | (string,optional) A textual description of the cause of this pipeline run. Will be set as cause of the Jenkins job. If null or empty, no cause information will be available. |
| `spec.logging` | (object,optional) The logging configuration. |
| `spec.logging.elasticsearch` | (object,optional) The configuration for pipeline logging to Elasticsearch. If not specified, logging to Elasticsearch is disabled and the default Jenkins log implementation is used (stdout of Jenkinsfile Runner container). |
| `spec.logging.elasticsearch.runID` | (any,optional) The JSON value that should be set as field `runId` in each log entry in Elasticsearch. It can be any JSON value (`null`, boolean, number, string, list, map). |
| `spec.timeout` | (string,optional) The timeout value specified for a steward pipeline run. The duration string format of composed of whole numbers, each with a unit suffix, such as "300m", "15h" or "2h45m". Valid time units are "s", "m" and "h". |


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
| `status.result` | (string,optional) The result code of the pipeline run as single-word string.<br/><br/> Possible values are:<ul><li>`success`: The pipeline run was processed successfully.</li><li>`error_infra`: The pipeline run failed due to an infrastructure problem.</li><li>`error_config`: The pipeline run failed due to a client-side configuration error in the `spec` section.</li><li>`error_content`: The pipeline run failed due to a content problem, or the cause of the failure could not be detected as an infrastructure problem (e.g. a network glitch breaking a pipeline step).</li><li>`aborted`: The pipeline run has been aborted.</li><li>`timeout`: The pipeline run exceeded the maximum execution time.</li></ul> |
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
