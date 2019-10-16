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

```bash
kubectl apply -f ./backend-k8s/steward-system
```

### Prepare Namespace for Back-End Client

**Example only:**
```bash
# edit yaml and apply
kubectl apply -f ./backend-k8s/steward-client-example
```

See the yaml files in this folder for more details about configuration options.


[tekton-install]: https://github.com/tektoncd/pipeline/blob/master/docs/install.md
