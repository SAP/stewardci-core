# Monitoring

Steward provides some metrics to be collected by [Prometheus] or compatible monitoring software.
There is also an [example dashboard][example-dashboard] for [Grafana] available to display the metrics.

## Steward Metrics

### Tenant Metrics

| Name | Type | Description |
| ---- | ---- | ----------- |
| `steward_tenants_total` | gauge | number of tenants in the cluster |

### Pipeline Run Metrics

| Name | Type | Label | Description |
| ---- | ---- | ----- | ----------- |
| `steward_pipelineruns_started_total`   | counter   | _none_ | counter is increased by every started pipeline run |
| `steward_pipelineruns_completed_total` | counter   | result | counters with result label are increased when result of pipeline run is set |
| `steward_pipelinerun_duration_seconds` | histogram | state  | histogram with 15 exponential buckets starting from 125ms with factor 2 for the different pipelinerun states |

## Example Installation with Prometheus Operator

### Prerequisites

-   Environment variable `$KUBECONFIG` set
-   Helm 3 installed locally and initialized
-   You have cloned this repo and and your current directory is `docs/monitoring`

### Install Prometheus Operator

[Prometheus Operator][prometheus-operator] is a common way to get Prometheus on a Kubernetes cluster.

There is a [Helm chart for Prometheus Operator][prometheus-operator-chart] that can be installed like this:

```bash
helm install monitoring stable/prometheus-operator \
    --namespace monitoring
```

See the [chart documentation][prometheus-operator-chart] for installation details.

### Install Steward service monitor resources for Prometheus Operator

The [Steward Helm chart](../../charts/steward/README.md) can create service monitor resources for Prometheus Operator.
By default this is disabled and can be enabled by parameter `metrics.serviceMonitors.enabled=true`.
See the [documentation of chart parameters `metrics.serviceMonitors.*`](../../charts/steward/README.md#monitoring) for details.

Service monitors can be enabled both for new installations and upgrades.
An upgrade can also be used if the installed Steward version should be kept.
In such case specify the same chart version that is installed, and just change the chart parameters.

### Install Grafana Dashboards for Steward

File [`docs/monitoring/grafana_dashboard`](./grafana_dashboard.json) in this repository contains the definition of a Grafana dashboard for Steward.

It can be added to the Grafana instance installed with Prometheus Operator:

```bash
kubectl -n monitoring create configmap monitoring-prometheus-oper-steward --from-file ./grafana_dashboard.json \
&& kubectl -n monitoring label configmap monitoring-prometheus-oper-steward grafana_dashboard=1
```

### Access the Grafana UI

To access Grafana with your local browser, establish a port-forwarding to the Grafana service on your cluster.
The following example forwards your local port 7900 to Grafana's service port:

```bash
kubectl -n monitoring port-forward $(kubectl -n monitoring --selector=app=grafana get pod -o name) 7900:3000
```

You can then access Grafana using this URL:

    http://localhost:7900/

You may choose another local port number according to your needs.


[example-dashboard]: grafana_dashboard.json
[Prometheus]: https://prometheus.io/docs/introduction/overview/
[Grafana]: https://grafana.com
[tiller-install]: https://rancher.com/docs/rancher/v2.x/en/installation/ha/helm-init/#install-tiller-on-the-cluster
[prometheus-operator]: https://github.com/coreos/prometheus-operator
[prometheus-operator-chart]: https://github.com/helm/charts/tree/master/stable/prometheus-operator
