# Troubleshooting

This page provides tips and tricks for troubleshooting

## Run Controller is unresponsive

When the run controller is unresponsive a thread dump can be triggered by
sending `SIGQUIT` to the corresponding processes:

```bash
pkill -QUIT -f 'steward-runctl'
```

For issuing the command above in a Kubernetes cluster a suitable pod needs to be launched, e.g.

```bash
kubectl run -i -t busybox \
  --image=busybox \
  --restart=Never \
  --overrides='{ "spec": { "hostPID" : true, "hostIPC" : false, "nodeSelector": { "<KEY>": "<VAL>" } } }'
```

With `hostPID` the pod container shares the host process ID namespace.

`nodeSelector` needs to be set accordingly in order to ensure the container for sending the signal resides on
the same node like the run-controller.

The logs can be accesses via:

```bash
kubectl -n steward-system logs -f "<POD>"
```
