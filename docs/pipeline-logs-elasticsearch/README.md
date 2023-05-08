# Sending Pipeline Logs to Elasticsearch

## Design

The [Jenkins Pipeline Elasticsearch Logs plug-in][jenkins-elasticsearch-logs] is used to stream logs, pipeline structure and progress to Elasticsearch.
It connects to the Elasticsearch instance directly, which requires to pass Elasticsearch credentials to the Jenkinsfile Runner.

__Passing Elasticsearch credentials to the Jenkinsfile Runner is a severe security issue.__
User-supplied code can access these credentials and use them to write arbitrary log entries to Elasticsearch, especially those that appear to belong to another pipeline run.

There is a single Elasticsearch instance used for all pipeline runs' logs, because pipeline-run-specific ElasticSeach instances are too expensive.
In addition, a single index will be used to hold logs of all pipeline runs, because Elasticsearch does not scale with hundreds or thousands of one-per-pipeline-run indices.

### Future Extensions

#### Use a log forwarder between Jenkins and Elasticsearch

-   Forwarder runs outside of the pipeline run sandbox.
-   Hold the actual Elasticsearch credentials.
-   Jenkins sends logs to forwarder, authenticating with temporary credentials used for the current pipeline run only.

#### Manage network connections with Istio

-   Use Istio to control the communication between log plug-in and forwarder
-   Benefits:
    -   plug-in talks to local Envoy proxy without any credentials
    -   Credentials are managed externally via Istio
    -   Rate limiting, retry, ...

## Configuration

Logging to Elasticsearch requires passing certain parameters as environment variables to the Jenkinsfile Runner container (`PIPELINE_LOG_ELASTICSEARCH_*`).
As we run the Jenkinsfile Runner container via Tekton, our Tekton Task sets those environment variables based on optional template parameters.
In the future all these parameters will be set by the Pipeline Run Controller based on configuration elsewhere.

For now only `PIPELINE_LOG_ELASTICSEARCH_RUN_ID_JSON` is set by the Pipeline Run Controller based on `spec.logging.elasticsearch.runID` of the respective PipelineRun resource.
In addition the Pipeline Run Controller sets `PIPELINE_LOG_ELASTICSEARCH_INDEX_URL` to the empty string if a PipelineRun resource does not specify `spec.logging.elasticsearch`.
Logging to Elasticsearch is disabled then and logs are written to the container's stdout.

### Enable logging to Elasticsearch

To enable a Steward instance to forward pipeline run logs to Elasticsearch, the index URL must be statically set in Steward's Task for the Jenkinsfile Runner.
The preferred way to do this is to specify the index URL as a parameter of the [Steward Helm chart](../../charts/steward/README.md).

## Testing

### Deploying Elasticsearch and Kibana in the Kubernetes cluster

For testing it might be sufficient to have Elasticsearch and Kibana running in the same Kubernetes cluster where pipelines are running.

Prerequisites:

-   Environment variable `$KUBECONFIG` set
-   Helm 2 or 3 (>= 2.14) installed locally
-   Helm initialized (For Helm2 [Tiller is installed][tiller-install] on the target cluster)

Add the Helm repo from Elastic:

```bash
$ helm repo add elastic https://helm.elastic.co
```

__Install Elasticsearch__ via the Helm chart from Elastic (not from `stable`):

```bash
$ helm install --name elasticsearch elastic/elasticsearch --version 7.3.0 \
    --namespace elasticsearch \
    --set replicas=2
```

`replicas` is the number of Elasticsearch nodes.
Due to pod anti-affinity rules these nodes run on different Kubernetes nodes.
Therefore your cluster will have multiple nodes running permanently.
Choosing a higher number here may increase your infrastructure costs.

With `replicas=1` we have seen that the status of the Elasticsearch cluster becomes yellow due to an unassigned shard.
As a consequence the Elasticsearch pod's readiness probe failed after a restart of the pod.

See the [Elasticsearch Helm chart documentation][elastic-elasticsearch-helm-chart] for more configuration options.

__Install Kibana__ via the Helm chart from Elastic (not from `stable`):

```bash
$ helm install --name kibana elastic/kibana --version 7.3.0 \
    --namespace kibana \
    --set elasticsearchHosts=http://elasticsearch-primary.elasticsearch.svc.cluster.local:9200
```

See the [Kibana Helm chart documentation][elastic-kibana-helm-chart] for more configuration options.

### Accessing the Kibana UI

To access Kibana with your local browser, establish a port-forwarding to the Kibana service on your cluster.
The following example forwards your local port 7800 to Kibana's service port:

```bash
$ kubectl port-forward -n kibana service/kibana-kibana 7800:5601
```

You can then access Kibana using this URL:

    http://localhost:7800/

You may choose another local port number according to your needs.


[elastic-elasticsearch-helm-chart]: https://github.com/elastic/helm-charts/tree/master/elasticsearch
[elastic-kibana-helm-chart]: https://github.com/elastic/helm-charts/tree/master/kibana
[jenkins-elasticsearch-logs]: https://github.com/SAP/elasticsearch-logs-plugin
[tiller-install]: https://rancher.com/docs/rancher/v2.x/en/installation/ha/helm-init/#install-tiller-on-the-cluster
