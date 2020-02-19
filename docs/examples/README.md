# Example Pipelines

This folder contains simple example `PipelineRun` resources which can be executed on your project "Steward" installation.
If you did not setup your project "Steward" or do not have access to a hosted instance please follow the [installation guide](../install/README.md).

## Tenant

Project "Steward" is designed to offer Pipeline-as-a-Service to many different tenants being completely isolated from each other (secrets, pipelines, logs, ...).

For each tenant a front-end client creates a tenant namespace. This is done by creating a `Tenant` resource in the clients namespace.

```sh
$ kubectl -n steward-c-client1 apply -f tenant.yaml
tenant.steward.sap.com/tenant1 created
```

To check the result execute:

```sh
kubectl -n steward-c-client1 get tenants.steward.sap.com
NAME                                   AGE     RESULT    TENANT-NAMESPACE
tenant1                                4m53s   success   steward-t-client1-tenant1-ga2xfm

```

*Note: A `Tenant` needs to be created only once per tenant.*

## PipelineRun

Now we can create a `PipelineRun` in the tenant namespace.

```sh
$ export TENANT_NAMESPACE=$(kubectl -n steward-c-client1 get tenants.steward.sap.com tenant1 -o=jsonpath={.status.tenantNamespaceName})
$ export RUN_NAME=$(kubectl -n $TENANT_NAMESPACE create -f pipelinerun_ok.yaml -o=name)
$ echo $RUN_NAME
pipelinerun.steward.sap.com/ok-md4kw
```

The status of the PipelineRun can be checked on the resource.

```sh
$ kubectl -n $TENANT_NAMESPACE get $RUN_NAME -owide
NAME       STARTED   FINISHED   STATUS    RESULT   MESSAGE
ok-md4kw   27s                  running            
```

The log can be found in the `step-jenkinsfile-runner` container of the runner pod in the temporarily created run namespace.

*Note: A better way is to [persist logs in Elasticsearch](../pipeline-logs-elasticsearch/README.md)*

*Note: You may use the pipelinerun_sleep.yaml in the create command above if you want to see the logs as described below.
The pipeline runs for 2 minutes before the run namespace with the pod is deleted.*
 
```sh
$ export RUN_NAMESPACE=$(kubectl -n $TENANT_NAMESPACE get $RUN_NAME -o=jsonpath={.status.namespace})
$ echo $RUN_NAMESPACE
$ export POD_NAME=$(kubectl -n $RUN_NAMESPACE get pod -o name)
$ echo $POD_NAME
$ kubectl -n $RUN_NAMESPACE logs $POD_NAME -c step-jenkinsfile-runner 
Cloning pipeline repository https://github.com/SAP-samples/stewardci-example-pipelines
Cloning into '.'...
Checking out pipeline from revision master
Your branch is up to date with 'origin/master'.
...
```
