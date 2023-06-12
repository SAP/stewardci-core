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

### Prepare a Namespace to Manage Pipeline Runs

It is recommended to use a dedicated namespace to manage Steward PipelineRun objects
and associated objects like Secrets.

K8s users or service accounts must have the respective privileges to work with Steward
PipelineRun and K8s Secret objects in the namespace.
Steward ships with a cluster role `steward-edit` that users and service accounts can
be bound to. It is also aggregated into cluster roles 'edit' and 'admin'.
However, using these predefined cluster roles is optional.
Permissions can also be granted by any other RBAC configuration.

## More

As a next step you might want to [test your project "Steward"](../examples/README.md) by running example pipelines.

[tekton-install]: https://github.com/tektoncd/pipeline/blob/master/docs/install.md
