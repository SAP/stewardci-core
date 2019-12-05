# Monitoring

Steward provides some metrics which can be collected with [Prometheus].
There is also an [example dashbord] for [Grafana] available to display the metrics.

## Tenant Controller Metrics

| name | type | description |
| ---- | ---- | ----------- |
| steward_tenant_total_number | gauge | number of tenants in the cluster |

## Pipelinerun Controller Metrics

| name | type | label | description |
| ---- | ---- | ----- | ----------- |
| steward_pipelineruns_started_total_count   | counter   | _none_ | counter is increased by every started pipeline run |
| steward_pipelineruns_completed_total_count | counter   | result | counters with result label are increased when result of pipeline run is set |
| steward_pipelinerun_duration_seconds | histogram | state  | histogram with 15 exponential buckets starting from 125ms with factor 2 for the different pipelinerun states |

## Example Installation with Prometheus Operator

### Prerequisites

-   Environment variable `$KUBECONFIG` set
-   Helm 2 (>= 2.14) installed locally
-   Helm initialized ([Tiller is installed][tiller-install] on the target cluster)
-   You have cloned this repo and are in the directory _docs/monitoring_

### Install Prometheus Operator
```
helm install --namespace monitoring --name monitoring stable/prometheus-operator
kubectl apply -f serviceMonitors
kubectl -n steward-system label serviceMonitor --all release=monitoring
```

### Install Grafana Dashboards
```
kubectl -n monitoring create configmap monitoring-prometheus-oper-steward --from-file grafana_dashboard.json
kubectl -n monitoring label configmap  monitoring-prometheus-oper-steward grafana_dashboard=1
kubectl -n monitoring --selector=app=grafana delete pod
```

### Establish Port Forwarding to Grafana
```
kubectl -n monitoring port-forward $(kubectl -n monitoring --selector=app=grafana get pod -o name) 3000:3000
```

[example dashbord]: grafana_dashboard.json
[Prometheus]: https://prometheus.io/docs/introduction/overview/
[Grafana]: https://grafana.com
[tiller-install]: https://rancher.com/docs/rancher/v2.x/en/installation/ha/helm-init/#install-tiller-on-the-cluster
