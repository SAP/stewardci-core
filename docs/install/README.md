# Installation Guide

To run you own project "Steward" you need a Kubernetes cluster.
Currently **Kubernetes 1.14** is recommended.

## Install Tekton v0.7.0

Project "Steward" requires Tekton. Please read the [Tekton installation instructions][tekton-install].

In short:

```bash
kubectl apply -f https://github.com/tektoncd/pipeline/releases/download/v0.7.0/release.yaml
```

## Install Steward

### Clone this repo

Clone the repo and change into the root directory, e.g.:

```bash
git clone "$THIS_REPO" stewardci-core
cd stewardci-core
```

### Install via Steward Helm Chart

See the [Steward Helm Chart documentation](../../charts/steward/README.md).

### Prepare Namespaces for Clients

Each Steward client gets its own client namespace to interact with Steward.

Typically a client is a (Web) application that uses Steward as pipeline execution engine.
Typically one Steward instance has only one client, but there can be any number of clients, e.g. in a test environment.

**Example only:**

```bash
# edit yaml and apply
kubectl apply -f ./backend-k8s/steward-client-example
```

See the yaml files in this folder for more details about configuration options.

## More

As a next step you might want to [test your project "Steward"](../examples/README.md) by running example pipelines.



[tekton-install]: https://github.com/tektoncd/pipeline/blob/master/docs/install.md
