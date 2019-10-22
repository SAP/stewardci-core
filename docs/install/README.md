# Installation Guide

To run you own project "Steward" you need a Kubernetes cluster. Currently **Kubernetes 1.13** is recommended.

## Install Tekton v0.7.0

Project "Steward" requires Tekton. Please read the [Tekton installation instructions][tekton-install].

In short:

```bash
kubectl apply -f https://github.com/tektoncd/pipeline/releases/download/v0.7.0/release.yaml
```

## Install the backend

### Clone this repo

Clone the repo and change into the backend-k8s directory:

```bash
git clone $THIS_REPO
```

### Create and Start Steward-System

*Note: If you want to store logs in Elasticsearch apply the changes described in [Sending Pipeline Logs to Elasticsearch](../pipeline-logs-elasticsearch/README.md) first.*

```bash
kubectl apply -f ./backend-k8s/steward-system
```

### Prepare Namespace for Back-End Client

Each front-end client gets an own back-end client namespace to operate with a project "Steward" instance. Typically a front-end client is a UI on top of our backend, and typically there is only one front-end client communicating with a project "Steward" instance.

**Example only:**
```bash
# edit yaml and apply
kubectl apply -f ./backend-k8s/steward-client-example
```

See the yaml files in this folder for more details about configuration options.


[tekton-install]: https://github.com/tektoncd/pipeline/blob/master/docs/install.md

### More

As a next step you might want to [test your project "Steward"](../examples/README.md) by running example pipelines.