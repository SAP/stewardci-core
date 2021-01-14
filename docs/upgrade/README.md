# Upgrade Guide

To upgrade steward you can set the system in the 'upgrade mode'.
In this mode running pipeline runs will be processed to the end.
Newly created pipeline runs will stay untouched.

## Steps

- Set the system in the 'upgrade mode'
- Wait until the already started pipeline runs are finished.
- Install the new version of steward
- Switch off 'upgrade mode'


### Set the system in the 'upgrade mode'

To set the system in the 'upgrade mode' you need to create a config map in the system namespace with specific content.
See the file 'upgrade_mode_on.yaml' in this directory as an example.

```bash
kubectl apply -n steward-system -f upgrade_mode_on.yaml
```

## Wait until the already started pipeline runs are finished.

You can list all non-finished pipelines with the following command
```bash
 kubectl get spr --all-namespaces | grep -v finished
 ```

Installation can start when there are only pipeline runs with state unknown.

### Install the new version of steward
See [Installing Steward](../install/README.md)

### Switch off 'upgrade mode'

To switch off the upgrade mode you can either delete the created config map or set `upgradeMode: "false"`

```bash
kubectl apply -n steward-system -f upgrade_mode_off.yaml
```

```bash
kubectl delete -n steward-system -f upgrade_mode_on.yaml
```