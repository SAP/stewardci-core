# Installation Guide

To run you own project "Steward" you need a Kubernetes cluster.
Currently **Kubernetes 1.24** is recommended.

## Install Tekton v0.41.1

Project "Steward" requires Tekton. Please read the [Tekton installation instructions][tekton-install].

In short:

```bash
kubectl apply -f https://github.com/tektoncd/pipeline/releases/download/v0.41.1/release.yaml
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

### Prepare Namespaces and Service Account for execution of Pipeline Runs

For the execution of Pipeline Runs a separate namespace should be created.
To be able to create Pipeline Runs in this namespace the corresponding ServiceAccount needs to be mapped to
the ClusterRole 'steward-edit' or the Default ClusterRole 'edit'. The ClusterRole 'steward-edit' is also
aggregated to the ClusterRole 'admin'.

To be able to create secrets in the new namespace the corresponding permissions need to
be granted to the service account separately.

## More

As a next step you might want to [test your project "Steward"](../examples/README.md) by running example pipelines.

[tekton-install]: https://github.com/tektoncd/pipeline/blob/master/docs/install.md
