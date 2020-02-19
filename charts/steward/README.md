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
| <code>pipelineRuns.<wbr/>logging.<wbr/>elasticsearch.<wbr/>indexURL</code> | (string)<br/> The URL of the Elasticsearch index to send logs to. If null or empty, logging to Elasticsearch is disabled. Example: `http://elasticsearch-master.elasticsearch.svc.cluster.local:9200/jenkins-logs/_doc` | empty |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>image.<wbr/>repository</code> | (string)<br/> The container registry and repository of the Jenkinsfile Runner image. | `stewardci/stewardci-jenkinsfile-runner` |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>image.<wbr/>tag</code> | (string)<br/> The tag of the Jenkinsfile Runner image in the container registry. | A fixed image tag. |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>image.<wbr/>pullPolicy</code> | (string)<br/> The image pull policy for the Tenant Controller image. For possible values see field `imagePullPolicy` of the `container` spec in the Kubernetes API documentation. | `IfNotPresent` |
| <code>pipelineRuns.<wbr/>jenkinsfileRunner.<wbr/>resources</code> | (object of [`RecourceRequirements`][k8s-resourcerequirements])<br/> The resource requirements of Jenkinsfile Runner containers. When overriding, override the complete value, not just subvalues, because the default value might change in future versions and a partial override might not make sense anymore. | Limits and requests set (see `values.yaml`) |
| <code>pipelineRuns.<wbr/>networkPolicy</code> | (string)<br/> The network policy to be created in every pipeline run namespace. The value must be a string containing a complete `networkpolicy.networking.k8s.io` resource manifest in YAML format. The `.metadata` section of the manifest can be omitted, as it will be replaced anyway. See the [Kubernetes documentation of network policies][k8s-networkpolicies] for details about Kubernetes network policies.<br/><br/> Note that Steward ensures that all pods in pipeline run namespaces are _isolated_ in terms of network policies. The policy defined here adds further egress and/or ingress rules. | A rule that allows ingress traffic from all pods in the same namespace and egress traffic to the internet, the cluster DNS and the Kubernetes API server. |

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
[prometheus-operator]: https://github.com/coreos/prometheus-operator
