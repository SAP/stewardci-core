# Secrets

This page describe the different kinds of secrets that are involved in a Steward setup.

- [Secrets](#secrets)
  - [Image Pull Secrets](#image-pull-secrets)
    - [Steward System Images](#steward-system-images)
    - [Jenkinsfile Runner Image](#jenkinsfile-runner-image)
    - [Pipeline Custom Pod Images](#pipeline-custom-pod-images)
  - [Git Server Authentication](#git-server-authentication)
    - [Pipeline Clone Secret](#pipeline-clone-secret)
    - [Source Code Repository Secrets](#source-code-repository-secrets)
  - [Jenkins Credentials](#jenkins-credentials)
  - [Other Secrets](#other-secrets)
    - [Log Storage in ElasticSearch](#log-storage-in-elasticsearch)
  - [Links](#links)


## Image Pull Secrets

Image pull secrets are required by Kubernetes to pull container images from private registries.
This section describes the different areas where image pull secrets are needed in a Steward setup.


### Steward System Images

If the images for Steward system pods reside in private registries, image pull secrets must be configured by a Steward operator.
By default the images are hosted in a public repository on Docker Hub, which does not require authentication when pulling.

Affected images:

- Tenant Controller pod
- Run Controller pod

The required image pull secrets must be created in namespace `steward-system` and attached to the `default` service account of that namespace.

More information on how to create image pull secrets in Kubernetes can be found in the Kubernetes documentation:

- [Pull an Image from a Private Registry][k8s_docs-pull_image_private_registry]
- [Add ImagePullSecrets to a service account][k8s_docs_add_imagepullsecrets_to_service_account]


### Jenkinsfile Runner Image

If the image of the Jenkinsfile Runner container resides in a private registry, an image pull secret must be configured by a Steward operator.
By default the Jenkinsfile Runner image is hosted in a public repository on Docker Hub, which does not require authentication when pulling.

__TODO__: How to configure the image pull secret for the Jenkinsfile Runner image


### Pipeline Custom Pod Images

Pipelines may start additional pods to execute steps in containers, either via the Jenkins Kubernetes plugin or by other means using the Kubernetes API.
If the required container images are hosted in private registries, image pull secrets must be provided by the respective Steward client (on behalf on end users).

A Steward PipelineRun resource object specifies the names of all image pull secrets required for that pipeline run:

```yaml
apiVersion: steward.sap.com/v1alpha1
kind: PipelineRun
spec:
    ...
    imagePullSecrets:
    - secret1
    - secret2
    - secret3
```

If no image pull secrets are required, `spec.imagePullSecrets` can be omitted or have an empty list value.

The given secrets are standard Kubernetes `v1/Secret` resource objects that must exist in the same namespace as the PipelineRun object that references them.
The secrets must be of the correct type (`kubernetes.io/dockerconfigjson`) to be usable for Kubernetes as image pull secret.
Besides that there are no further requirements like special annotations or labels.

When a pipeline gets executed in a transient sandbox namespace, the secrets listed in `spec.imagePullSecrets` of the corresponding PipelineRun resource object are copied to the sandbox namespace with a different name.
The Kubernetes service account of the Jenkinsfile Runner container has all those secrets attached as default image pull secrets.
Therefore, they will be used automatically when that service account creates pods based on images from private registries.
The pipeline and/or tools used by the pipeline to start additional pods are not required to specify image pull secrets in each pod specification, although this is still possible.

__:warning: Warning:__ Any code that gets executed by a pipeline AND has access to the Kubernetes service account token can read all image pull secrets! This is especially important to consider if untrusted code may get executed, e.g. a pipeline processing pull requests from untrusted users.

The following code has access to image pull secrets:

- Any code running in the Jenkinsfile Runner container, because this container has the service account token mounted.
- Any code running in additional containers that have the service account token mounted.

To prevent access to secrets, untrusted code must be executed in containers where the service account token will not be supplied to (mounting of service account token disabled via pod spec and token not passed into the container in any other way).

More information on how to create image pull secrets in Kubernetes can be found in the Kubernetes documentation:

- [Pull an Image from a Private Registry][k8s_docs-pull_image_private_registry]
- [Add ImagePullSecrets to a service account][k8s_docs_add_imagepullsecrets_to_service_account]


## Git Server Authentication

### Pipeline Clone Secret

Before Steward can run a pipeline, it needs to fetch the pipeline definition (Jenkinsfile) from a Git repository.
If cloning this Git repository requires authentication, a secret must be provided by the respective Steward client (on behalf of end users).

A Steward PipelineRun resource object can specify the secret to be used to authenticate at the Git server from where the pipeline definition should be taken:

```yaml
apiVersion: steward.sap.com/v1alpha1
kind: PipelineRun
spec:
    ...
    jenkinsFile:
        repoUrl: https://github.com/org1/pipelines
        ...
        repoAuthSecret: github-com-token1
```

If authentication is not required when cloning the pipeline repository, `spec.jenkinsFile.repoAuthSecret` can be omitted or set to an empty string value.

The value of `spec.jenkinsFile.repoAuthSecret` is the name of a Kubernetes `v1/Secret` resource object of type `kubernetes.io/basic-auth` that contains the username and password for authentication when cloning from `spec.jenkinsFile.repoUrl`.
Besides that there are no further requirements like special annotations or labels.

When a pipeline gets executed in a transient sandbox namespace, the pipeline clone secret specified in `spec.jenkinsFile.repoAuthSecret` of the corresponding PipelineRun resource object is copied to the sandbox namespace with a different name.
The Jenkinsfile Runner container has a generated Git credential file (`$HOME/.git-credentials`) that configures the username and password from that secret for the respective Git server.
This means that any further Git commands executed in the Jenkinsfile Runner container will use these credentials (for the respective Git server) if not explicitly overridden.

__:warning: Warning:__ Any code that gets executed by a pipeline AND runs in the Jenkinsfile Runner container or has access to the Kubernetes service account token can read the pipeline clone secret!
This is especially important to consider if untrusted code may get executed, e.g. a pipeline processing pull requests from untrusted users.

The following code has access to a pipeline sync secret:

- Any code running in the Jenkinsfile Runner container, because this container has the credentials in `$HOME/.git-credentials` and has the service account token mounted.
- Any code running in additional containers that have the service account token mounted.

To prevent access to secret, untrusted code must be executed in containers where the service account token will not be supplied to (mounting of service account token disabled via pod spec and token not passed into the container in any other way).


### Source Code Repository Secrets

Pipelines usually clone source from one or more Git repositories.
They may even need write access to them, for instance to create tags, send status feedback or push generated commits.
If this requires authentication, secrets must be provided by the respective Steward client (on behalf of end users).

In the special case where the __pipeline definition (Jenkinsfile) and the sources are located in the same repository__, only the [Pipeline Clone Secret](#pipeline-clone-secret) needs to be configured.
If the pipeline clone secret should be available as Jenkins credential, e.g. because the pipeline must fetch sources in a container other than the Jenkinsfile Runner container, the respective Kubernetes Secret resource object should have the required annotations (see [Jenkins Credentials](#jenkins-credentials) below).

In all other cases, credentials needed to access source code repositories have to be configured as [Jenkins Credentials](#jenkins-credentials) as described below.


## Jenkins Credentials

Pipelines typically need credentials of different kinds to access protected resources and services, e.g. source code repositories, artifact repositories and deployment targets.

Steward allows to define those credentials as regular Kubernetes `v1/Secret` resource objects and makes them available in Jenkins as regular Jenkins credentials with the help of the [Jenkins Kubernetes Credentials Provider Plugin][jenkins_k8s_credential_provider_plugin].
The secrets are provided by the respective Steward client (on behalf of end users).

A Steward PipelineRun resource object specifies the names of all secrets to be used as Jenkins credentials for that pipeline run:

```yaml
apiVersion: steward.sap.com/v1alpha1
kind: PipelineRun
spec:
    ...
    secrets:
    - secret1
    - secret2
    - secret3
```

If no Jenkins credentials are required, `spec.secrets` can be omitted or have an empty list value.

The given secrets are standard Kubernetes `v1/Secret` resource objects that must exist in the same namespace as the PipelineRun object that references them.

The Kubernetes Credentials Provider Plugin requires Secret resource objects to be of __certain types and carry special labels and annotations__ in order to map correctly them to Jenkins credential types.
Details can be found on the [Examples page][jenkins_k8s_credential_provider_plugin_examples] of the plugin.

When a pipeline gets executed in a transient sandbox namespace, the secrets listed in `spec.secrets` of the corresponding PipelineRun resource object are copied to the sandbox namespace with the same name.
It is also possible to rename the secret while it gets copied by providing the desired name as annotation `steward.sap.com/secret-rename-to` on the original secret. The desired name must be a valid Kubernetes Secret name and be unique within the sandbox namespace. In `spec.secrets` of pipeline runs the original secret name must be used to select secrets.
The Jenkins Kubernetes Credentials Provider Plugin will use the secrets from the sandbox namespace only.
Any secret that is not listed in `spec.secrets` will not be available as Jenkins credential.

__:warning: Warning:__ Any code that gets executed by a pipeline AND has access to the Kubernetes service account token can read all image pull secrets! This is especially important to consider if untrusted code may get executed, e.g. a pipeline processing pull requests from untrusted users.

The following code has access to Jenkins credential secrets:

- Any code running in the Jenkinsfile Runner container, because this container has the service account token mounted.
- Any code running in additional containers that have the service account token mounted.

To prevent access to secrets, untrusted code must be executed in containers where the service account token will not be supplied to (mounting of service account token disabled via pod spec and token not passed into the container in any other way).


## Other Secrets

### Log Storage in ElasticSearch

The log output of pipeline runs can be send to an Elasticsearch server.
The credentials to authenticate at the Elasticsearch server must be provided by the Steward client.

See documenation page [Sending Pipeline Logs to Elasticsearch](../pipeline-logs-elasticsearch/README.md).

__TODO:__ How to configure credentials for Elasticseach logging


## Links

- Kubernetes Secrets:
    - the concept of [Secrets][k8s_docs_secrets] (K8s docs)
    - the definition of secret types in the [source code][k8s_secret_types_src]
    - [Pull an Image from a Private Registry][k8s_docs-pull_image_private_registry] (K8s docs)
    - [Add ImagePullSecrets to a service account][k8s_docs_add_imagepullsecrets_to_service_account] (K8s docs)
    - [Distribute Credentials Securely Using Secrets][k8s_docs_distribute_credentials_secure] (K8s docs)

<p/>

- Jenkins Kubernetes Credentials Provider Plugin:
    - [Home Page][jenkins_k8s_credential_provider_plugin]
    - [Examples][jenkins_k8s_credential_provider_plugin_examples]



[jenkins_k8s_credential_provider_plugin]: https://jenkinsci.github.io/kubernetes-credentials-provider-plugin/
[jenkins_k8s_credential_provider_plugin_examples]: https://jenkinsci.github.io/kubernetes-credentials-provider-plugin/examples/
[k8s_docs-pull_image_private_registry]: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
[k8s_docs_add_imagepullsecrets_to_service_account]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#add-imagepullsecrets-to-a-service-account
[k8s_docs_secrets]: https://kubernetes.io/docs/concepts/configuration/secret/
[k8s_docs_distribute_credentials_secure]: https://kubernetes.io/docs/tasks/inject-data-application/distribute-credentials-secure/
[k8s_secret_types_src]: https://github.com/kubernetes/kubernetes/blob/e09f5c40b55c91f681a46ee17f9bc447eeacee57/pkg/apis/core/types.go#L4360-L4444
