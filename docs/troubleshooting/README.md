# Troubleshooting

This page provides tips and tricks for troubleshooting

## Tenant Controller or Run Controller are unresponsible

When either Run Controller or Tenant Controller are unresponsibe a thread dump can be triggered by
sending `SIGQUIT` to the corresponding processes:

```bash
pkill -QUIT -f '<PATTERN>'
```

where `PATTERN` depicts the application (`steward-runctl`,`steward-tenantctl`).

For issuing the command above in a kubernetes cluster a suitable pod needs to be launched, e.g.

```bash
kubectl run -i -t busybox --image=busybox --restart=Never --overrides='{ "spec": { "hostPID" : true, "hostIPC" : false } }'
```

With `hostPID` the pod contains shares the host process ID namespace.

The logs can be accesses via:

```bash
kubectl -n steward-system logs -f <POD>
```
