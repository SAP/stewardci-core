> Here you find a [basic overview](Secrets.md) on our secrets

# Secrets - Technical View

Kubernetes provides Secrets (`kind: Secret`) of various `types`. Below you find some relevant examples. See the [`Secret` go doc](https://github.com/kubernetes/kubernetes/blob/e09f5c40b55c91f681a46ee17f9bc447eeacee57/pkg/apis/core/types.go#L4360-L4444) and the [documentation](https://kubernetes.io/docs/concepts/configuration/secret/) for more details, e.g. which additional annotations are required.

- `type: Opaque` *(Should not be used in our scenario)*
  > is the default; arbitrary user-defined data
- `type: kubernetes.io/basic-auth`
  > contains data needed for basic authentication
- `type: kubernetes.io/ssh-auth`
  > contains data needed for SSH authentication
- `type: kubernetes.io/service-account-token`
  > contains a token that identifies a service account to the API
- `type: kubernetes.io/dockerconfigjson`
  > contains a dockercfg file that follows the same format rules as ~/.docker/config.json

Depending on where secrets are used later on additional annotations or labels are required.

**Tekton** for example supports and requires the following annotation(s). See [Tekton documentation](https://github.com/tektoncd/pipeline/blob/master/docs/auth.md) for more information.

```yaml
metadata:
  annotations:
    tekton.dev/git-0: https://github.com
```

**Jenkins Kubernetes Credentials Provider** on the other hand requires such an annotation and label:

```yaml
metadata:
  annotations:
    jenkins.io/credentials-description: Description displayed in Jenkins
  labels:
    jenkins.io/credentials-type: usernamePassword
```
