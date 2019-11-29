# monitoring

## install helm 2.16 or higher
```
wget https://get.helm.sh/helm-v2.16.1-linux-amd64.tar.gz
tar -zxvf helm-v2.16.1-linux-amd64.tar.gz
kubectl apply -f tiller.yaml
helm init
kubectl -n tiller get pod
## wait until pod has STATUS Running
helm version
```

## install prometheus operator
```
helm install --namespace monitoring --name monitoring stable/prometheus-operator
kubectl apply -f serviceMonitors
kubectl -n steward-system label serviceMonitor --all release=monitoring
```

## install grafana dashbords
```
kubectl -n monitoring create configmap monitoring-prometheus-oper-steward --from-file grafana_dashboard.json
kubectl -n monitoring label configmap  monitoring-prometheus-oper-steward grafana_dashboard=1
kubectl -n monitoring --selector=app=grafana delete pod
```

## establish port forwarding to grafana
```
kubectl -n monitoring port-forward $(kubectl -n monitoring --selector=app=grafana get pod -o name) 3000:3000
```

