# Steward Metrics Reference <!-- no-toc -->

- [Steward Metrics Reference](#steward-metrics-reference)
  - [General](#general)
    - [Retries](#retries)
      - [`steward_retries_retrycount`](#steward_retries_retrycount)
      - [`steward_retries_latency_seconds`](#steward_retries_latency_seconds)
  - [Kubernetes API Calls](#kubernetes-api-calls)
    - [REST Client](#rest-client)
      - [`steward_k8sclient_rest_ratelimit_latency_millis`](#steward_k8sclient_rest_ratelimit_latency_millis)
      - [`steward_k8sclient_rest_request_latency_millis`](#steward_k8sclient_rest_request_latency_millis)
      - [`steward_k8sclient_rest_request_results`](#steward_k8sclient_rest_request_results)
  - [Steward Pipeline Run Controller](#steward-pipeline-run-controller)
    - [Processing Indicators](#processing-indicators)
      - [`steward_pipelineruns_controller_heartbeats_total`](#steward_pipelineruns_controller_heartbeats_total)
      - [`steward_pipelineruns_started_total`](#steward_pipelineruns_started_total)
      - [`steward_pipelineruns_completed_total`](#steward_pipelineruns_completed_total)
      - [`steward_pipelineruns_state_duration_seconds`](#steward_pipelineruns_state_duration_seconds)
      - [DEPRECATED `steward_pipelinerun_state_duration_seconds`](#deprecated-steward_pipelinerun_state_duration_seconds)
      - [`steward_pipelineruns_ongoing_state_duration_periodic_observations_seconds`](#steward_pipelineruns_ongoing_state_duration_periodic_observations_seconds)
      - [DEPRECATED `steward_pipelinerun_ongoing_state_duration_periodic_observations_seconds`](#deprecated-steward_pipelinerun_ongoing_state_duration_periodic_observations_seconds)
      - [DEPRECTATED `steward_pipelinerun_update_seconds`](#deprectated-steward_pipelinerun_update_seconds)
    - [Run Controller Workqueue](#run-controller-workqueue)
      - [`steward_pipelineruns_workqueue_depth`](#steward_pipelineruns_workqueue_depth)
      - [`steward_pipelineruns_workqueue_adds_total`](#steward_pipelineruns_workqueue_adds_total)
      - [`steward_pipelineruns_workqueue_latency_seconds`](#steward_pipelineruns_workqueue_latency_seconds)
      - [`steward_pipelineruns_workqueue_workduration_seconds`](#steward_pipelineruns_workqueue_workduration_seconds)
      - [`steward_pipelineruns_workqueue_unfinished_workduration_seconds`](#steward_pipelineruns_workqueue_unfinished_workduration_seconds)
      - [`steward_pipelineruns_workqueue_longest_running_processor_seconds`](#steward_pipelineruns_workqueue_longest_running_processor_seconds)
      - [`steward_pipelineruns_workqueue_retry_count_total`](#steward_pipelineruns_workqueue_retry_count_total)
  - [Steward Tenant Controller](#steward-tenant-controller)
    - [Processing Indicators](#processing-indicators-1)
      - [`steward_tenants_controller_heartbeats_total`](#steward_tenants_controller_heartbeats_total)
      - [`steward_tenants_count_total`](#steward_tenants_count_total)
      - [DEPRECATED `steward_tenants_total`](#deprecated-steward_tenants_total)
    - [Tenant Controller Workqueue](#tenant-controller-workqueue)
      - [`steward_tenants_workqueue_depth`](#steward_tenants_workqueue_depth)
      - [`steward_tenants_workqueue_adds_total`](#steward_tenants_workqueue_adds_total)
      - [`steward_tenants_workqueue_latency_seconds`](#steward_tenants_workqueue_latency_seconds)
      - [`steward_tenants_workqueue_workduration_seconds`](#steward_tenants_workqueue_workduration_seconds)
      - [`steward_tenants_workqueue_unfinished_workduration_seconds`](#steward_tenants_workqueue_unfinished_workduration_seconds)
      - [`steward_tenants_workqueue_longest_running_processor_seconds`](#steward_tenants_workqueue_longest_running_processor_seconds)
      - [`steward_tenants_workqueue_retry_count_total`](#steward_tenants_workqueue_retry_count_total)


## General

### Retries

The Steward implementation contains various retry loops, e.g. for resilience or synchronization.
Those retry loops should be instrumented to emit some metrics that can help analyzing problems where processing takes longer as expected.

#### `steward_retries_retrycount`

Generic metric for retry loops collecting the number of retries performed for retried operations.

Labels:

| Name | Description |
|---|---|
| `location` | The retry loop's code location. This is typically a full-qualified function name. |


#### `steward_retries_latency_seconds`

Generic metric for retry loops collecting the latency (in seconds) caused by retrying operations.

Labels:

| Name | Description |
|---|---|
| `location` | The retry loop's code location. This is typically a full-qualified function name. |


## Kubernetes API Calls

### REST Client

Steward uses the Go module `k8s.io/client-go` to call Kubernetes APIs calls.
This library contains a generic REST client, that provides some metrics.

#### `steward_k8sclient_rest_ratelimit_latency_millis`

A histogram vector of client-side late limit latency partitioned by URL scheme, hostname, port, URL path and HTTP method.

Type: Histogram Vector

Labels:

| Name | Description |
|---|---|
| `scheme` | The URL scheme. |
| `hostname` | The destination hostname. |
| `port` | The destination port number. |
| `path` | The URL path. |
| `method` | The HTTP method. |

#### `steward_k8sclient_rest_request_latency_millis`

A histogram vector of request latency partitioned by URL scheme, hostname, port, URL path and HTTP method.

Type: Histogram Vector

Labels:

| Name | Description |
|---|---|
| `scheme` | The URL scheme. |
| `hostname` | The destination hostname. |
| `port` | The destination port number. |
| `path` | The URL path. |
| `method` | The HTTP method. |

#### `steward_k8sclient_rest_request_results`

The number of finished requests partitioned by host, HTTP method and status code.

Type: Counter Vector

Labels:

| Name | Description |
|---|---|
| `host` | The destination hostname. |
| `method` | The HTTP method. |
| `status` | The HTTP status code. |


## Steward Pipeline Run Controller

### Processing Indicators

#### `steward_pipelineruns_controller_heartbeats_total`

The number of heartbeats of the run controller instance.

Type: Counter


#### `steward_pipelineruns_started_total`

The total number of started pipeline runs.


#### `steward_pipelineruns_completed_total`

The number of completed pipeline runs partitioned by result type.

Labels:

| Name | Description |
|---|---|
| `result` | The pipeline run result type as defined in the Steward API. |


#### `steward_pipelineruns_state_duration_seconds`

A histogram vector partitioned by pipeline run states counting the pipeline runs that finished a state grouped by the state duration.

There's one histogram per pipeline run state (label `state`).
A pipeline run gets counted immediately when a state is finished.

Labels:

| Name | Description |
|---|---|
| `state` | The pipeline run state name as defined in the Steward API. |


#### DEPRECATED `steward_pipelinerun_state_duration_seconds`

**DEPRECATED**. Use `steward_pipelineruns_state_duration_seconds` instead.

Identical to `steward_pipelineruns_state_duration_seconds`.


#### `steward_pipelineruns_ongoing_state_duration_periodic_observations_seconds`

A histogram vector partitioned by pipeline run states that counts the number of periodic observations of pipeline runs in a state grouped by the duration of the state at the time of the observation.

The purpose of this metric is the detection of overly long processing times, caused by e.g. hanging controllers.

There's one histogram per pipeline run state (label `state`).
All existing pipeline runs get counted periodically, i.e. every observation cycle counts each pipeline run in exactly one histogram.
This means a single pipeline run is counted zero, one or multiple times in the same or different buckets of the same or different histograms.
This in turn means without knowing the observation and scraping intervals it is not possible to infer the _absolute_ number of pipeline runs observed.
It is only meaningful to calculate a _ratio_ between observations in certain buckets and the total number of observations (in a single or across multiple histograms).

Pipeline runs that are marked as deleted are not counted to exclude delays caused by finalization.

Labels:

| Name | Description |
|---|---|
| `state` | The pipeline run state name as defined in the Steward API. |


#### DEPRECATED `steward_pipelinerun_ongoing_state_duration_periodic_observations_seconds`

**DEPRECATED**. Use `steward_pipelineruns_ongoing_state_duration_periodic_observations_seconds` instead.

Identical to `steward_pipelineruns_ongoing_state_duration_periodic_observations_seconds`.


#### DEPRECTATED `steward_pipelinerun_update_seconds`

Deprecated. Use [REST Client metrics](#rest-client) and [retries metrics](#retries) instead.

A histogram vector of the duration of update operations.

Labels:

| Name | Description |
|---|---|
| `type` | An identifier of the update operation. Only `UpdateState` (updating a pipeline run status) was used. |


### Run Controller Workqueue

The Steward Run Controller has an in-memory workqueue of pipeline run objects to be processed.
Multiple concurrent worker threads fetch items from the workqueue for processing.
In case of errors workers may put items back into the queue to retry processing later (using a back-off delay).
Once an item is finished, it gets removed from the queue.

#### `steward_pipelineruns_workqueue_depth`

The current depth of the workqueue.

Type: Gauge


#### `steward_pipelineruns_workqueue_adds_total`

The number of entries _added_ to the workqueue over time.

Type: Counter


#### `steward_pipelineruns_workqueue_latency_seconds`

A histogram of queuing latency.
The latency is the time an item was waiting in the queue until processing the item started.
The processing time is therefore not included.

Type: Histogram


#### `steward_pipelineruns_workqueue_workduration_seconds`

A histogram of per-item _processing_ times.
The processing time of a queue item is the time the application _worked_ on it, but not the time it has been waiting in the queue.

Type: Histogram


#### `steward_pipelineruns_workqueue_unfinished_workduration_seconds`

The sum of processing time spent on items still in the queue.
Once an item gets removed from the workqueue, it does not count into this metric anymore.

Type: Gauge


#### `steward_pipelineruns_workqueue_longest_running_processor_seconds`

The longest processing time spent on a single item that is still in the queue.

Type: Gauge


#### `steward_pipelineruns_workqueue_retry_count_total`

The total number of retries needed to process queue items.

Type: Counter


## Steward Tenant Controller

### Processing Indicators

#### `steward_tenants_controller_heartbeats_total`

The number of heartbeats of the tenant controller instance.

Type: Counter


#### `steward_tenants_count_total`

The current number of tenants in the system.

Type: Counter


#### DEPRECATED `steward_tenants_total`

**DEPRECATED**. Use `steward_tenants_count_total` instead.

Identical to `steward_tenants_count_total`.


### Tenant Controller Workqueue

The Steward Tenant Controller has an in-memory workqueue of tenant objects to be processed.
Multiple concurrent worker threads fetch items from the workqueue for processing.
In case of errors workers may put items back into the queue to retry processing later (using a back-off delay).
Once an item is finished, it gets removed from the queue.

#### `steward_tenants_workqueue_depth`

The current depth of the workqueue.

Type: Gauge


#### `steward_tenants_workqueue_adds_total`

The number of entries _added_ to the workqueue over time.

Type: Counter


#### `steward_tenants_workqueue_latency_seconds`

A histogram of queuing latency.
The latency is the time an item was waiting in the queue until processing the item started.
The processing time is therefore not included.

Type: Histogram


#### `steward_tenants_workqueue_workduration_seconds`

A histogram of per-item _processing_ times.
The processing time of a queue item is the time the application _worked_ on it, but not the time it has been waiting in the queue.

Type: Histogram


#### `steward_tenants_workqueue_unfinished_workduration_seconds`

The sum of processing time spent on items still in the queue.

Type: Gauge


#### `steward_tenants_workqueue_longest_running_processor_seconds`

The longest processing time spent on a single item that is still in the queue.

Type: Gauge


#### `steward_tenants_workqueue_retry_count_total`

The total number of retries to process queue items.

Type: Counter
