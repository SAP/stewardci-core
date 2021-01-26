# Maintenance Guide

To maintain Steward you can put the system in _maintenance mode_.
In this mode running pipeline runs will be processed to the end.
Newly created pipeline runs will stay untouched.

## Steps

- [Put Steward in maintenance mode](#put-steward-in-maintenance-mode)
- [Wait until already started pipeline runs are finished](#wait-until-already-started-pipeline-runs-are-finished)
- [Install the new version of Steward](#install-the-new-version-of-steward)
- [Switch off 'maintenance mode'](#switch-off-maintenance-mode)


### Put Steward in maintenance mode

To put the system in maintenance mode you need to create a config map in the system namespace with specific content.
See the file 'maintenance_mode_on.yaml' in this directory as an example.

```bash
kubectl apply -n steward-system -f maintenance_mode_on.yaml
```

## Wait until the already started pipeline runs are finished.

You can list all non-finished pipelines with the following command:

```bash
 kubectl get spr --all-namespaces | grep -v finished
 ```

The installation can start when there are no pipeline runs at all or _all_ pipeline runs are in one of these states:

- <blank>
- finished

For each pipeline run without state you can also see an event with reason 'SkipOnMaintenanceMode':

```bash
kubectl get event -n  "$TENANT_NAMESPACE"
LAST SEEN   TYPE     REASON                  OBJECT                 MESSAGE
12s         Normal   SkipOnMaintenanceMode   pipelinerun/ok-n9lcl   Maintenance mode skip
```

### Install the new version of Steward
See [Installing Steward](../install/README.md)

### Switch off maintenance mode

To switch off the maintenance mode you can either delete the created config map:

```bash
kubectl delete -n steward-system -f maintenance_mode_on.yaml
```

or set `maintenanceMode: "false"`:

```bash
kubectl apply -n steward-system -f maintenance_mode_off.yaml
```
