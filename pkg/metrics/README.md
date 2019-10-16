
# Metrics

Our controllers expose metrics on port 9090. The corresponding services make those ports available.

To test locally you can forward the ports:

```sh
kubectl  -n steward-system port-forward steward-controller-758bf54c77-9qcxb 9090:9090 &
curl localhost:9090/metrics | grep steward
```

```sh
kubectl  -n steward-system port-forward steward-tenant-controller-... 9091:9090 &
curl localhost:9091/metrics | grep steward
```
 
