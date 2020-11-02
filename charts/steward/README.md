# Steward Helm Chart

Install and configure [Steward][] on Kubernetes.

## Table of Content

- [Steward Helm Chart](#steward-helm-chart)
  - [Table of Content](#table-of-content)
  - [Prerequisites](#prerequisites)
  - [Install](#install)
  - [Upgrade](#upgrade)
  - [Uninstall](#uninstall)
  - [Chart Configuration](#chart-configuration)
    - [Target Namespace](#target-namespace)
    - [Controllers](#controllers)
    - [Monitoring](#monitoring)
    - [Pipeline Runs](#pipeline-runs)
  - [Custom Resource Definitions](#custom-resource-definitions)

## Prerequisites

This Helm chart requires _Helm 3_ or higher.

The Steward Helm chart is currently not published in any public Helm repository.
Therefore it must be installed from a source checkout.

## Install

Use the `helm install` command to install the Steward Helm chart:

```bash
helm install RELEASE_NAME CHECKOUT_DIR/charts/steward/ ...
```

The `helm install` command has a parameter `--namespace` that defines the target namespace of the release.
Normally this is the namespace where the application will be installed to.
Helm also stores the release data in that namespace.
However, the Steward chart does not use the release namespace but has a separate parameter `targetNamespace.name` defining the namespace where Steward will be installed to.
This allows to include the Steward chart as dependency into another chart but still install into an own namespace.
The Helm release target namespace and the Steward target namespace can be equal if required.

If the Steward target namespace deliberately exists already, parameter `targetNamespace.create` should be set to `false` to suppress a resource conflict error.

Do not use the `--no-hooks` option of the `helm install` command.
Hooks are required for a consistent installation.

## Upgrade

Use the `helm upgrade` command to upgrade Steward releases:

```bash
helm upgrade RELEASE_NAME CHECKOUT_DIR/charts/steward/ ...
```

To reuse values from the current release revision, __do _NOT_ use the `--reuse-values` option__ of the `helm upgrade` command.
This option will not only reuse overridden values, but also the built-in values of the current release's chart version.
The result might be unexpected. Instead:

1.  Retrieve only the overridden values from the current release:

    ```bash
    helm get values RELEASE_NAME --namespace RELEASE_NAMESPACE --output yaml \
        >prev-values.yaml
    ```

2.  Apply the overridden values to the upgrade, optionally adding more overrides:

    ```bash
    helm upgrade ... -f prev-values.yaml -f new-values.yaml --set ...
    ```

    Note the order of increasing precedence from left to right!

## Uninstall

Use the `helm uninstall` command to delete a Steward release:

```bash
helm uninstall RELEASE_NAME ...
```

Note that Steward's custom resource definitions will not be deleted automatically (see [Custom Resource Definitions](#custom-resource-definitions) below).

## Chart Configuration

The tables in the following sections list the configurable parameters of the Steward chart.

### Target Namespace

| Parameter | Description | Default |
|---|---|---|
| <code>targetNamespace.<wbr/>create</code> | (bool)<br/> Whether to create the target namespace. Can be set to `false` if the namespace exists already, e.g. because the target namespace is also the target namespace of the Helm release and therefore must be created before installing the Chart. | `true` |
| <code>targetNamespace.<wbr/>name</code> | (string)<br/> The name of the namespace where Steward should be installed to. Note that we do not use the Helm release target namespace, so that this chart can be used as subchart of another chart and still installs into its dedicated namespace. | `steward-system` |

### Controllers

Pipeline Run Controller:

| Parameter | Description | Default |
|---|---|---|
| <code>runController.<wbr/>image.<wbr/>repository</code> | (string)<br/> The container registry and repository of the Run Controller image. | `stewardci/stewardci-run-controller` |
| <code>runController.<wbr/>image.<wbr/>tag</code> | (string)<br/> The tag of the Run Controller image in the container registry. | A fixed image tag. |
| <code>runController.<wbr/>image.<wbr/>pullPolicy</code> | (string)<br/> The image pull policy for the Run Controller image. For possible values see field `imagePullPolicy` of the `container` spec in the Kubernetes API documentation.  | `IfNotPresent` |
| <code>runController.<wbr/>resources</code> | (object of [`RecourceRequirements`][k8s-resourcerequirements])<br/> The resource requirements of the Run Controller container. When overriding, override the complete value, not just subvalues, because the default value might change in future versions and a partial override might not make sense anymore. | Limits and requests set (see `values.yaml`) |
| <code>runController.<wbr/>podSecurityContext</code> | (object of [`PodSecurityContext`][k8s-podsecuritycontext])<br/> The pod security context of the Run Controller pod. | `{}` |
| <code>runController.<wbr/>securityContext</code> | (object of [`SecurityContext`][k8s-securitycontext])<br/> The security context of the Run Controller container. | `{}` |
| <code>runController.<wbr/>nodeSelector</code> | (object)<br/> The `nodeSelector` field of the Run Controller [pod spec][k8s-podspec]. | `{}` |
| <code>runController.<wbr/>affinity</code> | (object of [`Affinity`][k8s-affinity])<br/> The `affinity` field of the Run Controller [pod spec][k8s-podspec]. | `{}` |
| <code>runController.<wbr/>tolerations</code> | (array of [`Toleration`][k8s-tolerations])<br/> The `tolerations` field of the Run Controller [pod spec][k8s-podspec]. | `[]` |
| <code>runController.<wbr/>args.<wbr/>qps</code> | (integer)<br/> The maximum queries per second (QPS) from the controller to the cluster. | 5 |
| <code>runController.<wbr/>args.<wbr/>burst</code> | (integer)<br/> The burst limit for throttle connections (maximum number of concurrent requests). | 10 |
| <code>runController.<wbr/>args.<wbr/>logVerbosity</code> | (integer)<br/> The log verbosity. Levels are adopted from [Kubernetes logging conventions][k8s-logging-conventions]. | 2 |

Tenant Controller:

| Parameter | Description | Default |
|---|---|---|
| <code>tenantController.<wbr/>image.<wbr/>repository</code> | (string)<br/> The container registry and repository of the Tenant Controller image. | `stewardci/stewardci-tenant-controller` |
| <code>tenantController.<wbr/>image.<wbr/>tag</code> | (string)<br/> The tag of the Tenant Controller image in the container registry. | A fixed image tag. |
| <code>tenantController.<wbr/>image.<wbr/>pullPolicy</code> | (string)<br/> The image pull policy for the Tenant Controller image. For possible values see field `imagePullPolicy` of the `container` spec in the Kubernetes API documentation.  | `IfNotPresent` |
| <code>tenantController.<wbr/>resources</code> | (object of [`RecourceRequirements`][k8s-resourcerequirements])<br/> The resource requirements of the Tenant Controller container. When overriding, override the complete value, not just subvalues, because the default value might change in future versions and a partial override might not make sense anymore. | Limits and requests set (see `values.yaml`) |
| <code>tenantController.<wbr/>podSecurityContext</code> | (object of [`PodSecurityContext`][k8s-podsecuritycontext])<br/> The pod security context of the Tenant Controller pod. | `{}` |
| <code>tenantController.<wbr/>securityContext</code> | (object of [`SecurityContext`][k8s-securitycontext])<br/> The security context of the Tenant Controller container. | `{}` |
| <code>tenantController.<wbr/>nodeSelector</code> | (object)<br/> The `nodeSelector` field of the Tenant Controller [pod spec][k8s-podspec]. | `{}` |
| <code>tenantController.<wbr/>affinity</code> | (object of [`Affinity`][k8s-affinity])<br/> The `affinity` field of the Tenant Controller [pod spec][k8s-podspec]. | `{}` |
| <code>tenantController.<wbr/>tolerations</code> | (array of [`Toleration`][k8s-tolerations])<br/> The `tolerations` field of the Tenant Controller [pod spec][k8s-podspec]. | `[]` |
| <code>tenantController.<wbr/>possibleTenantRoles</code> | (array of string)<br/> The names of all possible tenant roles. A tenant role is a Kubernetes ClusterRole that the controller binds within a tenant namespace to (a) the default service account of the client namespace the tenant belongs to and (b) to the default service account of the tenant namespace. The tenant role to be used can be configured per Steward client namespace via annotation `steward.sap.com/tenant-role`. | `['steward-tenant']` |
| <code>tenantController.<wbr/>args.<wbr/>logVerbosity</code> | The log verbosity. Levels are adopted from [Kubernetes logging conventions][k8s-logging-conventions]. | 2 |

Common parameters:

| Parameter | Description | Default |
|---|---|---|
| <code>imagePullSecrets</code> | (array of [LocalObjectReference][k8s-localobjectreference])<br/> The image pull secrets to be used for pulling controller images. | `[]` |

### Monitoring

| Parameter | Description | Default |
|---|---|---|
| <code>metrics.<wbr/>serviceMonitors.<wbr/>enabled</code> | (bool)<br/> Whether to generate ServiceMonitor resource for [Prometheus Operator][prometheus-operator]. | `false` |
| <code>metrics.<wbr/>serviceMonitors.<wbr/>extraLabels</code> | (object of string)<br/> Labels to be attached to the ServiceMonitor resources for [Prometheus Operator][prometheus-operator]. | `{}` |

### Pipeline Runs

| Parameter | Description | Default |
|---|---|---|
| <code>pipelineRuns.<wbr/>logging.<wbr/>elasticsearch.<wbr/>indexURL</code> | (string)<br/> The URL of the Elasticsearch index to send logs to. If null or empty, logging to Elasticsearch is disabled. Example: `http://elasticsearch-primary.elasticsearch.svc.cluster.local:9200/jenkins-logs/_doc` | empty |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>image.<wbr/>repository</code> | OUTDATED (string)<br/> Use <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>image</code> instead. | |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>image.<wbr/>tag</code> | OUTDATED (string)<br/> Use <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>image</code> instead.  | |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>image.<wbr/>pullPolicy</code> | OUTDATED (string)<br/> Use <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>imagePullPolicy</code> instead. | |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>image</code> | (string)<br/> The Jenkinsfile Runner image. | `stewardci/stewardci-jenkinsfile-runner:<versionTag>` |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>imagePullPolicy</code> | (string)<br/> The image pull policy for the Jenkinsfile Runner image. For possible values see field `imagePullPolicy` of the `container` spec in the Kubernetes API documentation. <br/><br/> **Currently broken, `IfNotPresent` is used in any case. See [tektoncd/pipeline #3423](https://github.com/tektoncd/pipeline/issues/3423)** | `IfNotPresent` |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>javaOpts</code> | (string)<br/> The JAVA_OPTS for the Jenkinsfile Runner process.  | (see `values.yaml`) |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>resources</code> | (object of [`RecourceRequirements`][k8s-resourcerequirements])<br/> The resource requirements of Jenkinsfile Runner containers. When overriding, override the complete value, not just subvalues, because the default value might change in future versions and a partial override might not make sense anymore. | Limits and requests set (see `values.yaml`) |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>podSecurityPolicy.<wbr/>runAsUser</code> | (integer)<br/> The user ID (UID) of the container processes of the Jenkinsfile Runner pod. The value must be an integer in the range of [1,65535]. Corresponds to field `runAsUser` of a [PodSecurityContext][k8s-podsecuritycontext]. | `1000` |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>podSecurityPolicy.<wbr/>runAsGroup</code> | (integer)<br/> The group ID (GID) of the container processes of the Jenkinsfile Runner pod. The value must be an integer in the range of [1,65535]. Corresponds to field `runAsGroup` of a [PodSecurityContext][k8s-podsecuritycontext]. | `1000` |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>podSecurityPolicy.<wbr/>fsGroup</code> | (integer)<br/> A special supplemental group ID of the container processes of the Jenkinsfile Runner pod, that defines the ownership of some volume types. The value must be an integer in the range of [1,65535]. Corresponds to field `fsGroup` of a [PodSecurityContext][k8s-podsecuritycontext]. | `1000` |
| <code>pipelineRuns.<wbr/>timeout</code> | (string)<br/> The maximum execution time of pipelines. Must be specified as a string understood by [Go's `time.parseDuration()`](https://godoc.org/time#ParseDuration): <blockquote>A duration string is a possibly signed sequence of decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".</blockquote> | `60m` |
| <code>pipelineRuns.<wbr/>networkPolicy</code> | (string)<br/> The network policy to be created in every pipeline run namespace. The value must be a string containing a complete `networkpolicy.networking.k8s.io` resource manifest in YAML format. The `.metadata` section of the manifest can be omitted, as it will be replaced anyway. See the [Kubernetes documentation of network policies][k8s-networkpolicies] for details about Kubernetes network policies.<br/><br/> Note that Steward ensures that all pods in pipeline run namespaces are _isolated_ in terms of network policies. The policy defined here adds further egress and/or ingress rules. | A rule that allows ingress traffic from all pods in the same namespace and egress traffic to the internet, the cluster DNS and the Kubernetes API server. |
| <code>pipelineRuns.<wbr/>limitRange</code> | (string)<br/> The limit range to be created in every pipeline run namespace. The value must be a string containing a complete `limitrange` resource manifest in YAML format. The `.metadata` section of the manifest can be omitted, as it will be replaced anyway. See the [Kubernetes documentation of limit ranges][k8s-limitranges] for details about Kubernetes limit ranges. | A limit range defining a default CPU request of 0.5 CPUs, a default CPU limit of 3 CPUs, a default memory request of 0.5 GiB and a default memory limit of 3 GiB.<br/><br/>This default limit range might change with newer releases of Steward. It is recommended to set an own limit range to avoid unexpected changes with Steward upgrades. |
| <code>pipelineRuns.<wbr/>resourceQuota</code> | (string)<br/> The resource quota to be created in every pipeline run namespace. The value must be a string containing a complete `resourcequotas` resource manifest in YAML format. The `.metadata` section of the manifest can be omitted, as it will be replaced anyway. See the [Kubernetes documentation of resource quotas][k8s-resourcequotas] for details about Kubernetes resource quotas.| none |

## Custom Resource Definitions

Steward extends Kubernetes by a set of _custom resources types_ like Tenant and PipelineRun.
The respective _custom resource definitions_ (CRDs) are handled in a special way:

-   Upon _install_, _upgrade_ and _rollback_ the CRDs will be created or updated to the version from this chart.

    CRDs that are not part of the Steward version to be installed, upgraded to or rolled back to will _NOT_ be deleted to prevent unexpected deletion of objects of those custom resource types.

-   An _uninstall_ will keep the CRDs to prevent unexpected deletion of objects of those custom resource types.

-   The `--force` option of the `helm upgrade` or `helm rollback` command, which enables replacement by delete and recreate, does _NOT_ apply to CRDs.

Operators may delete Steward CRDs manually after Steward has been uninstalled.
By doing so, all resource objects of those types will be removed by Kubernetes, too.



[Steward]: https://github.com/SAP/stewardci-core
[k8s-podspec]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.14/#podspec-v1-core
[k8s-resourcerequirements]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.14/#resourcerequirements-v1-core
[k8s-podsecuritycontext]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.14/#podsecuritycontext-v1-core
[k8s-securitycontext]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.14/#securitycontext-v1-core
[k8s-affinity]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.14/#affinity-v1-core
[k8s-tolerations]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.14/#toleration-v1-core
[k8s-localobjectreference]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.14/#localobjectreference-v1-core
[k8s-networkpolicies]: https://kubernetes.io/docs/concepts/services-networking/network-policies/
[k8s-limitranges]: https://kubernetes.io/docs/concepts/policy/limit-range/
[k8s-resourcequotas]: https://kubernetes.io/docs/concepts/policy/resource-quotas/
[k8s-logging-conventions]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md#logging-conventions
[prometheus-operator]: https://github.com/coreos/prometheus-operator
